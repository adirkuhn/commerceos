package integrations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/integrations"
)

func strp(s string) *string { return &s }

func TestNewSalesOrderOrigin_aShopOrderNeedsItsConnectionAndExternalID(t *testing.T) {
	for _, origin := range []integrations.Origin{integrations.OriginWooCommerce, integrations.OriginBol} {
		o, err := integrations.NewSalesOrderOrigin(integrations.SalesOrderOriginInput{
			SalesOrderID: "so-1", Origin: origin, ConnectionID: strp("conn-1"), ExternalOrderID: "26217", ExternalKey: "wc_order_x",
		})
		require.NoError(t, err)
		assert.Equal(t, origin, o.Origin)
		assert.Equal(t, "conn-1", *o.ConnectionID)
		assert.Equal(t, "26217", o.ExternalOrderID)
		assert.Equal(t, "wc_order_x", o.ExternalKey)

		_, err = integrations.NewSalesOrderOrigin(integrations.SalesOrderOriginInput{SalesOrderID: "so-1", Origin: origin, ExternalOrderID: "26217"})
		assert.ErrorIs(t, err, integrations.ErrSalesOrderOriginInvalid, "%s without a connection", origin)
		_, err = integrations.NewSalesOrderOrigin(integrations.SalesOrderOriginInput{SalesOrderID: "so-1", Origin: origin, ConnectionID: strp("conn-1")})
		assert.ErrorIs(t, err, integrations.ErrSalesOrderOriginInvalid, "%s without its external id", origin)
	}
}

func TestNewSalesOrderOrigin_aPOSSaleNeedsItsExternalIDButNoConnection(t *testing.T) {
	o, err := integrations.NewSalesOrderOrigin(integrations.SalesOrderOriginInput{SalesOrderID: "so-1", Origin: integrations.OriginPOS, ExternalOrderID: "pos-42"})
	require.NoError(t, err)
	assert.Nil(t, o.ConnectionID)

	_, err = integrations.NewSalesOrderOrigin(integrations.SalesOrderOriginInput{SalesOrderID: "so-1", Origin: integrations.OriginPOS})
	assert.ErrorIs(t, err, integrations.ErrSalesOrderOriginInvalid)
}

func TestNewSalesOrderOrigin_anOrderMadeHereHasNoExternalReference(t *testing.T) {
	for _, origin := range []integrations.Origin{integrations.OriginManual, integrations.OriginWhatsApp, integrations.OriginInternal} {
		o, err := integrations.NewSalesOrderOrigin(integrations.SalesOrderOriginInput{SalesOrderID: "so-1", Origin: origin})
		require.NoError(t, err)
		assert.Equal(t, origin, o.Origin)
		assert.False(t, o.IsExternal())

		_, err = integrations.NewSalesOrderOrigin(integrations.SalesOrderOriginInput{SalesOrderID: "so-1", Origin: origin, ExternalOrderID: "x"})
		assert.ErrorIs(t, err, integrations.ErrSalesOrderOriginInvalid, "%s cannot carry an external id", origin)
	}
}

func TestNewSalesOrderOrigin_rejectsAnUnknownOriginOrMissingOrder(t *testing.T) {
	_, err := integrations.NewSalesOrderOrigin(integrations.SalesOrderOriginInput{SalesOrderID: "so-1", Origin: "shopify"})
	assert.ErrorIs(t, err, integrations.ErrSalesOrderOriginInvalid)
	_, err = integrations.NewSalesOrderOrigin(integrations.SalesOrderOriginInput{Origin: integrations.OriginManual})
	assert.ErrorIs(t, err, integrations.ErrSalesOrderOriginInvalid)
}
