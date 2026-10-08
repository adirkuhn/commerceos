package channels_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func pendingConfirmation() channelsDomain.BolShipmentConfirmation {
	return channelsDomain.BolShipmentConfirmation{
		ID: "conf-1", SalesOrderID: "so-1", ProcessStatusID: "ps-1",
		Status: channelsDomain.BolConfirmationPending,
	}
}

func TestBolShipmentConfirmation_resolveSuccess(t *testing.T) {
	c := pendingConfirmation()
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

	require.NoError(t, c.Resolve(channelsDomain.BolConfirmationSuccess, "", now))

	assert.Equal(t, channelsDomain.BolConfirmationSuccess, c.Status)
	assert.Empty(t, c.ErrorMessage)
	require.NotNil(t, c.ResolvedAt)
	assert.Equal(t, now, *c.ResolvedAt)
}

func TestBolShipmentConfirmation_resolveFailure_keepsBolsReason(t *testing.T) {
	c := pendingConfirmation()

	require.NoError(t, c.Resolve(channelsDomain.BolConfirmationFailure, "Invalid track and trace", time.Now()))

	assert.Equal(t, channelsDomain.BolConfirmationFailure, c.Status)
	assert.Equal(t, "Invalid track and trace", c.ErrorMessage)
}

func TestBolShipmentConfirmation_resolveToPending_isRejected(t *testing.T) {
	c := pendingConfirmation()

	err := c.Resolve(channelsDomain.BolConfirmationPending, "", time.Now())

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidBolConfirmationStatus)
}

func TestBolShipmentConfirmation_resolveTwice_isRejected(t *testing.T) {
	c := pendingConfirmation()
	require.NoError(t, c.Resolve(channelsDomain.BolConfirmationSuccess, "", time.Now()))

	err := c.Resolve(channelsDomain.BolConfirmationFailure, "late", time.Now())

	assert.ErrorIs(t, err, channelsDomain.ErrBolConfirmationAlreadyResolved)
	assert.Equal(t, channelsDomain.BolConfirmationSuccess, c.Status)
}

func TestParseBolConfirmationStatus(t *testing.T) {
	cases := map[string]channelsDomain.BolConfirmationStatus{
		"PENDING": channelsDomain.BolConfirmationPending,
		"SUCCESS": channelsDomain.BolConfirmationSuccess,
		"FAILURE": channelsDomain.BolConfirmationFailure,
		"TIMEOUT": channelsDomain.BolConfirmationTimeout,
		"success": channelsDomain.BolConfirmationSuccess,
	}
	for in, want := range cases {
		got, err := channelsDomain.ParseBolConfirmationStatus(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}

	_, err := channelsDomain.ParseBolConfirmationStatus("WEIRD")
	assert.ErrorIs(t, err, channelsDomain.ErrInvalidBolConfirmationStatus)
}

func TestBolShipmentConfirmation_needsManualAction(t *testing.T) {
	for status, want := range map[channelsDomain.BolConfirmationStatus]bool{
		channelsDomain.BolConfirmationPending: false,
		channelsDomain.BolConfirmationSuccess: false,
		channelsDomain.BolConfirmationFailure: true,
		channelsDomain.BolConfirmationTimeout: true,
	} {
		c := channelsDomain.BolShipmentConfirmation{Status: status}
		assert.Equal(t, want, c.NeedsManualAction(), string(status))
	}
}

func TestBolShipmentConfirmation_toldBol(t *testing.T) {
	for status, want := range map[channelsDomain.BolConfirmationStatus]bool{
		channelsDomain.BolConfirmationPending: true,
		channelsDomain.BolConfirmationSuccess: true,
		channelsDomain.BolConfirmationFailure: false,
		channelsDomain.BolConfirmationTimeout: false,
	} {
		c := channelsDomain.BolShipmentConfirmation{Status: status}
		assert.Equal(t, want, c.ToldBol(), string(status))
	}
}
