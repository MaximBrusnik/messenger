package email

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
)

// Service sends verification emails over SMTP.
type Service struct {
	host    string
	port    string
	user    string
	pass    string
	from    string
	appURL  string
	enabled bool
}

func NewService(host, port, user, pass, from, appURL string) *Service {
	return &Service{
		host:    host,
		port:    port,
		user:    user,
		pass:    pass,
		from:    from,
		appURL:  appURL,
		enabled: host != "" && user != "",
	}
}

func (s *Service) SendVerificationEmail(to string, token string) {
	if !s.enabled {
		log.Printf("email: verification service disabled; would send to %s link=%s/verify/token=%s", to, s.appURL, token)
		return
	}
	subject := "Subject: Подтверждение почты MessangerMax\r\n"
	body := fmt.Sprintf("Перейдите по ссылке для подтверждения:\r\n%s/api/v1/auth/verify-email?token=%s\r\n", strings.TrimRight(s.appURL, "/"), token)
	msg := []byte(subject + "MIME-version: 1.0;\r\nContent-Type: text/plain; charset=\"UTF-8\";\r\n\r\n" + body)
	addr := net.JoinHostPort(s.host, s.port)
	var err error
	if s.port == "465" {
		err = sendMailSSL(addr, smtp.PlainAuth("", s.user, s.pass, s.host), s.from, []string{to}, msg)
	} else {
		err = smtp.SendMail(addr, smtp.PlainAuth("", s.user, s.pass, s.host), s.from, []string{to}, msg)
	}
	if err != nil {
		log.Printf("email: failed to send to %s: %v", to, err)
	}
}

func sendMailSSL(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	host, _, _ := net.SplitHostPort(addr)
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	if err = c.Auth(auth); err != nil {
		return err
	}
	if err = c.Mail(from); err != nil {
		return err
	}
	for _, t := range to {
		if err = c.Rcpt(t); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write(msg); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func GenerateVerificationToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Printf("email: token generation error: %v", err)
	}
	return hex.EncodeToString(b)
}
