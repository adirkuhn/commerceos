//go:build postnl_live

package postnl_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/postnl"
)

// TestLive_sandboxCreateShipmentAndStatus runs the adapter against PostNL's
// real sandbox. Opt-in only:
//
//	POSTNL_API_KEY=... POSTNL_CUSTOMER_CODE=... POSTNL_CUSTOMER_NUMBER=... \
//	  go test -tags postnl_live ./infrastructure/postnl/ -run Live -v
func TestLive_sandboxCreateShipmentAndStatus(t *testing.T) {
	creds := integrationsDomain.CarrierCredentials{
		Provider: integrationsDomain.ProviderPostNL, Sandbox: true,
		APIKey: os.Getenv("POSTNL_API_KEY"), CustomerCode: os.Getenv("POSTNL_CUSTOMER_CODE"), CustomerNumber: os.Getenv("POSTNL_CUSTOMER_NUMBER"),
	}
	if creds.APIKey == "" {
		t.Skip("POSTNL_API_KEY not set")
	}
	client, err := postnl.NewCarrierClientFactory().NewClient(creds)
	require.NoError(t, err)

	in := shipmentRequest()
	in.Sender = &integrationsDomain.CarrierAddress{Company: "Test Shop", Street: "Siriusdreef", Number: "42", PostalCode: "2132WT", City: "Hoofddorp", CC: "NL", Email: "shop@example.com"}
	result, err := client.CreateShipment(context.Background(), in)
	require.NoError(t, err)
	t.Logf("reference %s barcode %s label %d bytes", result.ExternalShipmentID, result.Barcode, len(result.Label.Data))
	assert.NotEmpty(t, result.Barcode)
	require.NotNil(t, result.Label)
	assert.True(t, bytes.HasPrefix(result.Label.Data, []byte("%PDF")), "a real PDF")

	tt, err := client.GetTrackTrace(context.Background(), result.ExternalShipmentID)
	require.NoError(t, err)
	t.Logf("status %d %s", tt.StatusCode, tt.StatusMain)
	assert.NotEmpty(t, tt.StatusMain)
}

func TestLive_sandboxEUShipmentWithReturnAddress(t *testing.T) {
	creds := integrationsDomain.CarrierCredentials{
		Provider: integrationsDomain.ProviderPostNL, Sandbox: true,
		APIKey: os.Getenv("POSTNL_API_KEY"), CustomerCode: os.Getenv("POSTNL_CUSTOMER_CODE"), CustomerNumber: os.Getenv("POSTNL_CUSTOMER_NUMBER"),
	}
	if creds.APIKey == "" {
		t.Skip("POSTNL_API_KEY not set")
	}
	client, err := postnl.NewCarrierClientFactory().NewClient(creds)
	require.NoError(t, err)

	in := shipmentRequest()
	in.Sender = &integrationsDomain.CarrierAddress{Company: "Test Shop", Street: "Siriusdreef", Number: "42", PostalCode: "2132WT", City: "Hoofddorp", CC: "NL", Email: "shop@example.com"}
	in.ReturnAddress = &integrationsDomain.CarrierAddress{Company: "Test Shop Returns", Street: "Siriusdreef", Number: "42", PostalCode: "2132WT", City: "Hoofddorp", CC: "NL"}
	in.Recipient = integrationsDomain.CarrierAddress{Person: "Max Mustermann", Street: "Invalidenstrasse", Number: "10", PostalCode: "10115", City: "Berlin", CC: "DE", Email: "max@example.com"}
	result, err := client.CreateShipment(context.Background(), in)
	require.NoError(t, err)
	t.Logf("EU reference %s barcode %s label %d bytes", result.ExternalShipmentID, result.Barcode, len(result.Label.Data))
	require.NotNil(t, result.Label)
	assert.True(t, bytes.HasPrefix(result.Label.Data, []byte("%PDF")))
}
