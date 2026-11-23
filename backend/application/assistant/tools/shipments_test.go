package tools_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	appsearch "github.com/adirkuhn/commerceos/application/search"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

type stubShipments struct {
	out   []shippingDomain.Shipment
	err   error
	gotID string
}

func (s *stubShipments) Execute(_ context.Context, salesOrderID string) ([]shippingDomain.Shipment, error) {
	s.gotID = salesOrderID
	return s.out, s.err
}

type stubTracking struct {
	byShipment map[string][]shippingDomain.ShipmentStatusEvent
	err        error
}

func (s *stubTracking) Execute(_ context.Context, shipmentID string) ([]shippingDomain.ShipmentStatusEvent, error) {
	return s.byShipment[shipmentID], s.err
}

type stubBolConfirmations struct {
	out []channelsDomain.BolShipmentConfirmation
	err error
}

func (s *stubBolConfirmations) Execute(context.Context, string) ([]channelsDomain.BolShipmentConfirmation, error) {
	return s.out, s.err
}

func TestGetOrderShipmentsTool(t *testing.T) {
	label := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	scan := label.Add(5 * time.Hour)
	delivered := label.Add(28 * time.Hour)
	cancelled := label.Add(-time.Hour)
	shipments := &stubShipments{out: []shippingDomain.Shipment{
		{ID: "sh-1", SalesOrderID: orderUUID, Carrier: shippingDomain.CarrierPostNL, Barcode: "3SABC123", StatusMain: "delivered",
			TrackingURL: "https://jouw.postnl.nl/track/3SABC123", PackageType: shippingDomain.PackageTypePackage, RecipientPhone: "+31600000000", CreatedAt: label},
		{ID: "sh-0", SalesOrderID: orderUUID, Carrier: shippingDomain.CarrierDHLForYou, CancelledAt: &cancelled, CancelNote: "wrong carrier", CreatedAt: cancelled},
	}}
	tracking := &stubTracking{byShipment: map[string][]shippingDomain.ShipmentStatusEvent{"sh-1": {
		{StatusMain: "registered", OccurredAt: label},
		{StatusMain: "en_route", OccurredAt: scan},
		{StatusMain: "delivered", OccurredAt: delivered},
	}}}
	resolved := label.Add(6 * time.Hour)
	bol := &stubBolConfirmations{out: []channelsDomain.BolShipmentConfirmation{
		{Action: "create_shipment", TransporterCode: "TNT", TrackAndTrace: "3SABC123", Status: "succeeded", CreatedAt: scan, ResolvedAt: &resolved},
	}}
	search := &stubSearch{out: &appsearch.GlobalSearchResult{SalesOrders: []omsDomain.SalesOrder{{ID: orderUUID, Number: "CO-12"}}}}
	tool := tools.NewGetOrderShipmentsTool(shipments, tracking, bol, search)
	assertStrictDefinition(t, tool, "get_order_shipments")

	got := call(t, tool, `{"order":"CO-12"}`)

	assert.Equal(t, orderUUID, shipments.gotID)
	assert.Equal(t, "/sales-orders/"+orderUUID, got["admin_path"])
	list := got["shipments"].([]any)
	require.Len(t, list, 2)
	first := list[0].(map[string]any)
	assert.Equal(t, "PostNL", first["carrier"])
	assert.Equal(t, "3SABC123", first["tracking_number"])
	assert.Equal(t, "https://jouw.postnl.nl/track/3SABC123", first["tracking_url"])
	assert.Equal(t, "delivered", first["status"])
	assert.Equal(t, "2026-11-20T09:00:00Z", first["label_created_at"])
	assert.Equal(t, "2026-11-20T14:00:00Z", first["first_carrier_scan_at"])
	assert.Equal(t, "2026-11-21T13:00:00Z", first["delivered_at"])
	assert.Equal(t, true, first["active"])
	assert.NotContains(t, first, "recipient_phone", "the recipient's contact details stay out")
	second := list[1].(map[string]any)
	assert.Equal(t, false, second["active"])
	assert.Equal(t, "wrong carrier", second["cancel_note"])

	confirmations := got["bol_confirmations"].([]any)
	require.Len(t, confirmations, 1)
	assert.Equal(t, "succeeded", confirmations[0].(map[string]any)["status"])
	assert.Equal(t, "3SABC123", confirmations[0].(map[string]any)["track_and_trace"])
}

func TestGetOrderShipmentsTool_noShipmentsYet(t *testing.T) {
	tool := tools.NewGetOrderShipmentsTool(&stubShipments{}, &stubTracking{}, &stubBolConfirmations{}, &stubSearch{})

	got := call(t, tool, `{"order":"`+orderUUID+`"}`)

	assert.Equal(t, []any{}, got["shipments"])
	assert.NotContains(t, got, "bol_confirmations")
}

func TestGetOrderShipmentsTool_errors(t *testing.T) {
	tool := func(s *stubShipments, tr *stubTracking, b *stubBolConfirmations) *tools.GetOrderShipmentsTool {
		return tools.NewGetOrderShipmentsTool(s, tr, b, &stubSearch{})
	}
	one := []shippingDomain.Shipment{{ID: "sh-1"}}

	assert.ErrorIs(t, callErr(t, tool(&stubShipments{}, &stubTracking{}, &stubBolConfirmations{}), `{}`), tools.ErrInvalidArguments)
	assert.ErrorContains(t, callErr(t, tool(&stubShipments{err: errors.New("db down")}, &stubTracking{}, &stubBolConfirmations{}), `{"order":"`+orderUUID+`"}`), "db down")
	assert.ErrorContains(t, callErr(t, tool(&stubShipments{out: one}, &stubTracking{err: errors.New("events down")}, &stubBolConfirmations{}), `{"order":"`+orderUUID+`"}`), "events down")
	assert.ErrorContains(t, callErr(t, tool(&stubShipments{}, &stubTracking{}, &stubBolConfirmations{err: errors.New("bol down")}), `{"order":"`+orderUUID+`"}`), "bol down")
	assert.ErrorContains(t, callErr(t, tool(&stubShipments{}, &stubTracking{}, &stubBolConfirmations{}), `{"order":"CO-99"}`), "no order numbered")
}
