package noop_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/infrastructure/messaging/noop"
)

func TestClient_Send_alwaysFailsLoudly(t *testing.T) {
	err := noop.Client{}.Send(context.Background(), messaging.Message{Channel: messaging.ChannelEmail, To: "shopper@example.com"})

	require.ErrorContains(t, err, "email sending is not configured")
}
