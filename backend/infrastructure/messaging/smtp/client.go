// Package smtp implements domain/messaging.Client over stdlib net/smtp --
// points at any SMTP relay (Gmail/Workspace, Mailgun/SES's SMTP endpoint,
// a self-hosted Postfix, etc.) via plain config, no vendor SDK required.
// A future REST-API provider (Resend, SendGrid...) would live alongside
// this as its own domain/messaging.Client implementation, selected the
// same way infrastructure/myparcel/infrastructure/stripe's own factories
// are.
package smtp

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/adirkuhn/commerceos/domain/messaging"
)

// Config is intentionally plain fields, not a struct tag/env-loader magic
// (see this repo's config-loading convention) -- internal/config populates
// it from SMTP_HOST/SMTP_PORT/SMTP_USERNAME/SMTP_PASSWORD/SMTP_FROM.
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type Client struct {
	cfg Config
}

func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg}
}

var _ messaging.Client = (*Client)(nil)

// Send uses net/smtp.SendMail, which opportunistically STARTTLS's when the
// server advertises the extension (true of virtually every relay on port
// 587) -- no implicit-TLS (port 465/SMTPS) support, matching what a
// generic STARTTLS relay needs; that's an explicit gap, not silently
// assumed to work.
func (c *Client) Send(_ context.Context, msg messaging.Message) error {
	if msg.Channel != messaging.ChannelEmail {
		return fmt.Errorf("smtp.Client.Send: unsupported channel %q", msg.Channel)
	}

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
		return fmt.Errorf("smtp.Client.Send: %w", err)
	}
	return nil
}

// ClientFactory implements domain/messaging.EmailClientFactory -- see that
// interface's doc comment for why a factory exists even though there's
// only one Client implementation.
type ClientFactory struct{}

func NewClientFactory() ClientFactory { return ClientFactory{} }

func (ClientFactory) NewClient(host, port, username, password, from string) messaging.Client {
	return NewClient(Config{Host: host, Port: port, Username: username, Password: password, From: from})
}
