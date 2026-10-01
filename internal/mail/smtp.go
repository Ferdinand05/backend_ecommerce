package mail

import (
	"context"
	"fmt"
	"net/smtp"
)

type SMTPConfig struct {
	Host            string
	Port            string
	Username        string
	Password        string
	From            string
	AppBaseURL      string
	FrontendBaseURL string
}

type SMTPSender struct {
	cfg SMTPConfig
}

func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{
		cfg: cfg,
	}
}

func (s *SMTPSender) SendVerificationEmail(
	ctx context.Context,
	to string,
	token string,
) error {
	verifyURL := fmt.Sprintf(
		"%s/auth/email-verification/verify?token=%s",
		s.cfg.FrontendBaseURL,
		token,
	)

	subject := "Verify your email"

	body := fmt.Sprintf(
		"Hello,\n\n"+
			"Please verify your email by opening this link:\n\n"+
			"%s\n\n"+
			"This verification link will expire soon.\n\n"+
			"Thank you.",
		verifyURL,
	)

	return s.send(to, subject, body)
}

func (s *SMTPSender) SendPasswordResetEmail(
	ctx context.Context,
	to string,
	token string,
) error {
	resetURL := fmt.Sprintf(
		"%s/reset-password?token=%s",
		s.cfg,
		token,
	)

	subject := "Reset your password"

	body := fmt.Sprintf(
		"Hello,\n\n"+
			"Please reset your password using this link:\n\n"+
			"%s\n\n"+
			"If you did not request this, you can ignore this email.",
		resetURL,
	)

	return s.send(to, subject, body)
}

func (s *SMTPSender) send(
	to string,
	subject string,
	body string,
) error {
	addr := fmt.Sprintf(
		"%s:%s",
		s.cfg.Host,
		s.cfg.Port,
	)

	message := []byte(
		"From: " + s.cfg.From + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			body,
	)

	var auth smtp.Auth

	if s.cfg.Username != "" {
		auth = smtp.PlainAuth(
			"",
			s.cfg.Username,
			s.cfg.Password,
			s.cfg.Host,
		)
	}

	return smtp.SendMail(
		addr,
		auth,
		s.cfg.From,
		[]string{to},
		message,
	)
}
