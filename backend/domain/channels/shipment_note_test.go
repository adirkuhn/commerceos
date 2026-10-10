package channels_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/locale"
)

func TestShipmentNote(t *testing.T) {
	assert.Equal(t, "Shipped via PostNL, tracking 3S1 - https://t/3S1", channels.ShipmentNote(locale.English, "PostNL", "3S1", "https://t/3S1"))
	assert.Equal(t, "Verzonden met PostNL, track & trace 3S1 - https://t/3S1", channels.ShipmentNote(locale.Dutch, "PostNL", "3S1", "https://t/3S1"))
	assert.Equal(t, "Enviado via PostNL, rastreio 3S1", channels.ShipmentNote(locale.Portuguese, "PostNL", "3S1", ""))
	assert.Equal(t, "Shipped via DHL, tracking X", channels.ShipmentNote("", "DHL", "X", ""))
}
