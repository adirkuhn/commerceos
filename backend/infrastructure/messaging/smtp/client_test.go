package smtp_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/infrastructure/messaging/smtp"
)

// TestClient_Send_wrapsConnectionFailure proves Send never silently
// swallows an unreachable/misconfigured relay -- see domain/messaging.Client's
// doc comment on why that matters (no outbox/retry for messaging today).
func TestClient_Send_wrapsConnectionFailure(t *testing.T) {
	client := smtp.NewClient(smtp.Config{Host: "127.0.0.1", Port: "1", From: "no-reply@example.com"})

	err := client.Send(context.Background(), messaging.Message{
		Channel: messaging.ChannelEmail, To: "shopper@example.com", Subject: "Test", HTMLBody: "<p>hi</p>",
	})

	require.Error(t, err)
}

func TestClient_Send_rejectsNonEmailChannel(t *testing.T) {
	client := smtp.NewClient(smtp.Config{Host: "127.0.0.1", Port: "1", From: "no-reply@example.com"})

	err := client.Send(context.Background(), messaging.Message{Channel: messaging.ChannelSMS, To: "+3100000000", Body: "hi"})

	require.ErrorContains(t, err, "unsupported channel")
}
