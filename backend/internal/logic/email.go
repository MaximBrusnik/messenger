package logic

import (
	"crypto/rand"
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

	subject := "Подтвердите email — MessangerMax"
	body := fmt.Sprintf(`Здравствуйте!

Спасибо за регистрацию в MessangerMax.

Для подтверждения email-адреса перейдите по ссылке:
%s

Если вы не регистрировались в MessangerMax, просто проигнорируйте это письмо.

С уважением,
Команда MessangerMax`, link)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body)

	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.pass, s.host)

	if err := smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("email send failed: %w", err)
	}

	log.Printf("Verification email sent to %s", to)
	return nil
}

func GenerateVerificationToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
