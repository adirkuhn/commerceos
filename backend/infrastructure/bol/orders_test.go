package bol_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/bol"
)

// bolV10OrderJSON is GET /retailer/orders/{orderId} shaped exactly like
// Bol's Retailer API v10 Order schema (OrderOrderItem.fulfilment =
// OrderFulfilment{method, distributionParty, latestDeliveryDate,
// exactDeliveryDate, expiryDate, timeFrameType}), values from the spec's
// own examples.
const bolV10OrderJSON = `{
  "orderId": "A2K8290LP8",
  "pickupPoint": false,
  "orderPlacedDateTime": "2017-02-09T12:39:48+01:00",
  "shipmentDetails": {"firstName": "Hans", "surname": "de Grote", "streetName": "Skywalkerstraat", "houseNumber": "21", "zipCode": "1234AB", "city": "PLATOONDORP", "countryCode": "NL", "email": "x@example.com"},
  "orderItems": [
    {
      "orderItemId": "2012345678",
      "cancellationRequest": false,
      "fulfilment": {"method": "FBR", "distributionParty": "RETAILER", "latestDeliveryDate": "2017-02-10", "expiryDate": "2017-02-13", "timeFrameType": "REGULAR"},
      "offer": {"offerId": "6ff736b5", "reference": "MijnOffer0021"},
      "product": {"ean": "0000007740404", "title": "Product Title"},
      "quantity": 10, "quantityShipped": 0, "quantityCancelled": 0,
      "unitPrice": 12.99, "totalPrice": 129.9, "discounts": [], "commission": 5.18,
      "latestChangedDateTime": "2017-02-09T13:39:48+01:00"
    },
    {
      "orderItemId": "2012345679",
      "cancellationRequest": false,
      "fulfilment": {"method": "FBR", "distributionParty": "RETAILER", "latestDeliveryDate": null, "exactDeliveryDate": "2017-02-13", "timeFrameType": "APPOINTMENT"},
      "product": {"ean": "0000007740405", "title": "Other"},
      "quantity": 1, "quantityShipped": 0, "quantityCancelled": 0,
      "unitPrice": 1.0, "totalPrice": 1.0, "discounts": [], "commission": 0.1
    },
    {
      "orderItemId": "2012345680",
      "cancellationRequest": false,
      "product": {"ean": "0000007740406", "title": "No fulfilment"},
      "quantity": 1, "quantityShipped": 0, "quantityCancelled": 0,
      "unitPrice": 1.0, "totalPrice": 1.0, "discounts": [], "commission": 0.1
    }
  ]
}`

func TestAdapter_GetOrder_readsFulfilmentDeliveryDates(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			writeToken(w)
		case "/orders/A2K8290LP8":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(bolV10OrderJSON))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	o, err := bol.NewAdapter(client).GetOrder(t.Context(), "A2K8290LP8")

	require.NoError(t, err)
	require.Len(t, o.OrderItems, 3)
	require.NotNil(t, o.OrderItems[0].LatestDeliveryDate)
	assert.Equal(t, time.Date(2017, 2, 10, 0, 0, 0, 0, time.UTC), *o.OrderItems[0].LatestDeliveryDate)
	assert.Nil(t, o.OrderItems[0].ExactDeliveryDate)
	assert.Nil(t, o.OrderItems[1].LatestDeliveryDate)
	require.NotNil(t, o.OrderItems[1].ExactDeliveryDate)
	assert.Equal(t, time.Date(2017, 2, 13, 0, 0, 0, 0, time.UTC), *o.OrderItems[1].ExactDeliveryDate)
	assert.Nil(t, o.OrderItems[2].LatestDeliveryDate)
	assert.Nil(t, o.OrderItems[2].ExactDeliveryDate)
	assert.Equal(t, int64(1299), o.OrderItems[0].UnitPriceMinorUnits)
}

func TestAdapter_GetOrder_malformedDeliveryDateIsAnError(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			writeToken(w)
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"orderId":"X","orderItems":[{"orderItemId":"1","quantity":1,"fulfilment":{"method":"FBR","latestDeliveryDate":"10-02-2017"}}]}`))
		}
	})

	_, err := bol.NewAdapter(client).GetOrder(t.Context(), "X")

	require.Error(t, err)
}
