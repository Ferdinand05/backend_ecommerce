package mail

import (
	"context"
	"fmt"
	"log/slog"
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

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Verify your email</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6;">
	<h2>Verify your email</h2>

	<p>Hello,</p>

	<p>
		Thank you for registering. Please verify your email address
		by clicking the button below.
	</p>

	<p>
		<a href="%s"
		   style="
				display: inline-block;
				padding: 12px 24px;
				background-color: #2563eb;
				color: #ffffff;
				text-decoration: none;
				border-radius: 6px;
				font-weight: bold;
		   ">
			Verify Email
		</a>
	</p>

	<p>
		This verification link will expire in 30 minutes.
	</p>

	<p>
		If you did not create an account, you can safely ignore this email.
	</p>

	<p>
		Thank you.
	</p>
</body>
</html>
`, verifyURL)

	return s.sendHTML(to, subject, htmlBody)
}

func (s *SMTPSender) SendPasswordResetEmail(
	ctx context.Context,
	to string,
	token string,
) error {
	resetURL := fmt.Sprintf(
		"%s/reset-password?token=%s",
		s.cfg.FrontendBaseURL,
		token,
	)

	subject := "Reset your password"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Reset your password</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6;">
	<h2>Reset your password</h2>

	<p>Hello,</p>

	<p>
		We received a request to reset your password.
	</p>

	<p>
		Click the button below to continue.
	</p>

	<p>
		<a href="%s"
		   style="
				display: inline-block;
				padding: 12px 24px;
				background-color: #2563eb;
				color: #ffffff;
				text-decoration: none;
				border-radius: 6px;
				font-weight: bold;
		   ">
			Reset Password
		</a>
	</p>

	<p>
		This reset link will expire in 30 minutes.
	</p>

	<p>
		If you did not request a password reset, you can safely ignore this email.
	</p>
</body>
</html>
`, resetURL)

	return s.sendHTML(to, subject, htmlBody)
}

func (s *SMTPSender) sendHTML(
	to string,
	subject string,
	htmlBody string,
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
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			htmlBody,
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

	if err := smtp.SendMail(
		addr,
		auth,
		s.cfg.From,
		[]string{to},
		message,
	); err != nil {
		slog.Error("mail.send_failed", "error", err)
		return err
	}

	return nil
}
