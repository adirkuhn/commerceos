package email_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	emailDomain "github.com/adirkuhn/commerceos/domain/email"
	"github.com/adirkuhn/commerceos/infrastructure/email"
)

// TestSMTPClient_Send_wrapsConnectionFailure proves Send never silently
// swallows an unreachable/misconfigured relay -- see domain/email.Client's
// doc comment on why that matters (no outbox/retry for email today).
func TestSMTPClient_Send_wrapsConnectionFailure(t *testing.T) {
	client := email.NewSMTPClient(email.SMTPConfig{Host: "127.0.0.1", Port: "1", From: "no-reply@example.com"})

	err := client.Send(context.Background(), emailDomain.Message{To: "shopper@example.com", Subject: "Test", HTMLBody: "<p>hi</p>"})

	require.Error(t, err)
}
