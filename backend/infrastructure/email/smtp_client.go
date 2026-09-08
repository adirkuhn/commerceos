// Package email implements domain/email.Client. SMTPClient is the only
// implementation today (stdlib net/smtp, no third-party dependency) --
// points at any SMTP relay (Gmail/Workspace, Mailgun/SES's SMTP endpoint,
// a self-hosted Postfix, etc.) via plain config, no vendor SDK required.
// A future REST-API provider (Resend, SendGrid...) would live alongside
// this as its own Client implementation, selected the same way
// infrastructure/myparcel/infrastructure/stripe's own factories are.
package email

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	emailDomain "github.com/adirkuhn/commerceos/domain/email"
)

// SMTPConfig is intentionally plain fields, not a struct tag/env-loader
// magic (see this repo's config-loading convention) -- internal/config
// populates it from SMTP_HOST/SMTP_PORT/SMTP_USER/SMTP_PASSWORD/SMTP_FROM.
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type SMTPClient struct {
	cfg SMTPConfig
}

func NewSMTPClient(cfg SMTPConfig) *SMTPClient {
	return &SMTPClient{cfg: cfg}
}

var _ emailDomain.Client = (*SMTPClient)(nil)

// Send uses net/smtp.SendMail, which opportunistically STARTTLS's when the
// server advertises the extension (true of virtually every relay on port
// 587) -- no implicit-TLS (port 465/SMTPS) support, matching what a
// generic STARTTLS relay needs; that's an explicit gap, not silently
// assumed to work.
func (c *SMTPClient) Send(_ context.Context, msg emailDomain.Message) error {
	addr := net.JoinHostPort(c.cfg.Host, c.cfg.Port)
	var auth smtp.Auth
	if c.cfg.Username != "" {
		auth = smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.Host)
	}

	headers := map[string]string{
		"From":         c.cfg.From,
		"To":           msg.To,
		"Subject":      msg.Subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=UTF-8",
	}
	var b strings.Builder
	for k, v := range headers {
		fmt.Fprintf(&b, "%s: %s\r\n", k, v)
	}
	b.WriteString("\r\n")
	b.WriteString(msg.HTMLBody)

	if err := smtp.SendMail(addr, auth, c.cfg.From, []string{msg.To}, []byte(b.String())); err != nil {
		return fmt.Errorf("smtpClient.Send: %w", err)
	}
	return nil
}
