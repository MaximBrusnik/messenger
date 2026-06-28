package logic

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"net/smtp"
)

type EmailService interface {
	SendVerificationEmail(to, token string) error
}

type emailService struct {
	host    string
	port    string
	user    string
	pass    string
	from    string
	appURL  string
	enabled bool
}

func NewEmailService(host, port, user, pass, from, appURL string) EmailService {
	enabled := host != "" && user != ""
	if enabled {
		log.Println("Email service enabled (SMTP:", host+")")
	} else {
		log.Println("Email service disabled — set SMTP_HOST and SMTP_USER to enable")
	}
	return &emailService{
		host:    host,
		port:    port,
		user:    user,
		pass:    pass,
		from:    from,
		appURL:  appURL,
		enabled: enabled,
	}
}

func (s *emailService) SendVerificationEmail(to, token string) error {
	if !s.enabled {
		log.Printf("[EMAIL DISABLED] Would send verification to %s with token %s", to, token)
		return nil
	}

	link := fmt.Sprintf("%s/verify-email?token=%s", s.appURL, token)

	subject := "Подтвердите email — MessAnger"
	body := fmt.Sprintf(`Здравствуйте!

Спасибо за регистрацию в MessAnger.

Для подтверждения email-адреса перейдите по ссылке:
%s

Если вы не регистрировались в MessangerMax, просто проигнорируйте это письмо.

С уважением,
Команда MessangerMax`, link)

	fromHeader := fmt.Sprintf("MessAnger <%s>", s.from)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		fromHeader, to, subject, body)

	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.pass, s.host)

	var err error
	if s.port == "465" {
		err = s.sendMailSSL(addr, auth, s.from, []string{to}, []byte(msg))
	} else {
		err = smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
	}
	if err != nil {
		return fmt.Errorf("email send failed: %w", err)
	}

	log.Printf("Verification email sent to %s", to)
	return nil
}

func (s *emailService) sendMailSSL(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	host := s.host
	tlsCfg := &tls.Config{ServerName: host}

	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}

	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	defer w.Close()

	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}

	return nil
}

func GenerateVerificationToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
