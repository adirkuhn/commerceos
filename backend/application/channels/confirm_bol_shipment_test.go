package channels_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeBolConfirmationRepo struct {
	existing []channelsDomain.BolShipmentConfirmation
	pending  []channelsDomain.BolShipmentConfirmation
	listErr  error
	created  []channelsDomain.BolShipmentConfirmation
	saved    []channelsDomain.BolShipmentConfirmation
	saveErr  error
}

func (f *fakeBolConfirmationRepo) Create(_ context.Context, c *channelsDomain.BolShipmentConfirmation) error {
	c.ID = fmt.Sprintf("conf-%d", len(f.created)+1)
	f.created = append(f.created, *c)
	return nil
}
func (f *fakeBolConfirmationRepo) ListBySalesOrder(context.Context, string) ([]channelsDomain.BolShipmentConfirmation, error) {
	return f.existing, f.listErr
}
func (f *fakeBolConfirmationRepo) ListPending(context.Context, int) ([]channelsDomain.BolShipmentConfirmation, error) {
	return f.pending, f.listErr
}
func (f *fakeBolConfirmationRepo) SaveResolution(_ context.Context, c *channelsDomain.BolShipmentConfirmation) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = append(f.saved, *c)
	return nil
}

type fakeBolClientSource struct {
	client       integrationsDomain.BolClient
	err          error
	connectionID []string
}

func (f *fakeBolClientSource) ClientFor(_ context.Context, connectionID string) (integrationsDomain.BolClient, error) {
	f.connectionID = append(f.connectionID, connectionID)
	return f.client, f.err
}

// ── fixtures ──────────────────────────────────────────────────────────────────

var confirmNow = time.Date(2026, 10, 7, 14, 11, 0, 0, time.UTC)

func shippedBolOrder() *omsDomain.SalesOrder {
	connID := "bol-conn-1"
	return &omsDomain.SalesOrder{
		ID: "so-1", TenantID: "tenant-1", Source: omsDomain.SalesOrderSourceBol, ExternalOrderID: "C000HLRF87",
		ConnectionID: &connID, Status: omsDomain.SalesOrderStatusShipped,
	}
}

func postNLShipment() shippingDomain.Shipment {
	return shippingDomain.Shipment{
		ID: "ship-1", SalesOrderID: "so-1", Carrier: shippingDomain.CarrierPostNL,
		Barcode: "3SMYPA859699114", RecipientCC: "NL", StatusMain: "handed_to_carrier",
	}
}

func openBolOrder(items ...integrationsDomain.BolOrderItem) func(context.Context, string) (*integrationsDomain.BolOrder, error) {
	return func(_ context.Context, orderID string) (*integrationsDomain.BolOrder, error) {
		return &integrationsDomain.BolOrder{OrderID: orderID, OrderItems: items}, nil
	}
}

type confirmFixture struct {
	order         *omsDomain.SalesOrder
	shipments     []shippingDomain.Shipment
	confirmations *fakeBolConfirmationRepo
	bol           *fakeBolClient
	source        *fakeBolClientSource
	outbox        *eventrecordtest.Outbox
}

func newConfirmFixture() *confirmFixture {
	bol := &fakeBolClient{getOrderFn: openBolOrder(integrationsDomain.BolOrderItem{OrderItemID: "item-1", Quantity: 1})}
	return &confirmFixture{
		order:         shippedBolOrder(),
		shipments:     []shippingDomain.Shipment{postNLShipment()},
		confirmations: &fakeBolConfirmationRepo{},
		bol:           bol,
		source:        &fakeBolClientSource{client: bol},
		outbox:        &eventrecordtest.Outbox{},
	}
}

func (f *confirmFixture) run(t *testing.T) error {
	t.Helper()
	uc := appchannels.NewConfirmBolShipmentUseCase(
		newMockGormDB(t, 3),
		&stubOrdersSalesOrderRepo{findByIDResult: f.order},
		&stubShipmentRepo{bySalesOrder: map[string][]shippingDomain.Shipment{"so-1": f.shipments}},
		f.confirmations, f.source, f.outbox, func() time.Time { return confirmNow },
	)
	return uc.Execute(context.Background(), "so-1")
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestConfirmBolShipment_shippedOrder_sendsShipmentToBolAndTracksIt(t *testing.T) {
	f := newConfirmFixture()
	f.bol.getOrderFn = openBolOrder(
		integrationsDomain.BolOrderItem{OrderItemID: "item-1", Quantity: 2},
		integrationsDomain.BolOrderItem{OrderItemID: "item-2", Quantity: 1},
	)
	f.bol.createShipmentFn = func(context.Context, integrationsDomain.BolShipmentInput) (integrationsDomain.BolProcessStatus, error) {
		return integrationsDomain.BolProcessStatus{ID: "ps-77", Status: "PENDING"}, nil
	}

	require.NoError(t, f.run(t))

	assert.Equal(t, []string{"bol-conn-1"}, f.source.connectionID)
	require.Len(t, f.bol.createShipmentCalls, 1)
	assert.Equal(t, integrationsDomain.BolShipmentInput{
		OrderItems: []integrationsDomain.BolShipmentItem{
			{OrderItemID: "item-1", Quantity: 2},
			{OrderItemID: "item-2", Quantity: 1},
		},
		ShipmentReference: "C000HLRF87",
		TransporterCode:   "TNT",
		TrackAndTrace:     "3SMYPA859699114",
	}, f.bol.createShipmentCalls[0])

	require.Len(t, f.confirmations.created, 1)
	c := f.confirmations.created[0]
	assert.Equal(t, channelsDomain.BolShipmentConfirmation{
		ID: "conf-1", TenantID: "tenant-1", SalesOrderID: "so-1", ConnectionID: "bol-conn-1", ShipmentID: "ship-1",
		ExternalOrderID: "C000HLRF87", Action: channelsDomain.BolActionCreateShipment, ProcessStatusID: "ps-77", TransporterCode: "TNT",
		TrackAndTrace: "3SMYPA859699114", Status: channelsDomain.BolConfirmationPending, CreatedAt: confirmNow,
	}, c)

	event, err := f.outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "channels.bol_shipment_confirmation.sent", event.Subject)
	assert.Equal(t, "conf-1", event.Payload["entity_id"])
	data := event.Data()
	assert.Equal(t, "so-1", data["sales_order_id"])
	assert.Equal(t, "pending", data["status"])
	assert.Equal(t, "ps-77", data["process_status_id"])
	assert.Equal(t, "TNT", data["transporter_code"])
	assert.Equal(t, "create_shipment", data["action"])
}

func TestConfirmBolShipment_bolAnswersSynchronously_storesFinalResult(t *testing.T) {
	f := newConfirmFixture()
	f.bol.createShipmentFn = func(context.Context, integrationsDomain.BolShipmentInput) (integrationsDomain.BolProcessStatus, error) {
		return integrationsDomain.BolProcessStatus{ID: "ps-77", Status: "SUCCESS"}, nil
	}

	require.NoError(t, f.run(t))

	require.Len(t, f.confirmations.created, 1)
	assert.Equal(t, channelsDomain.BolConfirmationSuccess, f.confirmations.created[0].Status)
	require.NotNil(t, f.confirmations.created[0].ResolvedAt)
}

func TestConfirmBolShipment_partlyShippedOrCancelledItems_sendsOnlyWhatIsStillOpen(t *testing.T) {
	f := newConfirmFixture()
	f.bol.getOrderFn = openBolOrder(
		integrationsDomain.BolOrderItem{OrderItemID: "done", Quantity: 1, QuantityShipped: 1},
		integrationsDomain.BolOrderItem{OrderItemID: "cancelled", Quantity: 1, QuantityCancelled: 1},
		integrationsDomain.BolOrderItem{OrderItemID: "partly", Quantity: 3, QuantityShipped: 1, QuantityCancelled: 1},
	)

	require.NoError(t, f.run(t))

	require.Len(t, f.bol.createShipmentCalls, 1)
	assert.Equal(t, []integrationsDomain.BolShipmentItem{{OrderItemID: "partly", Quantity: 1}}, f.bol.createShipmentCalls[0].OrderItems)
}

func TestConfirmBolShipment_nothingOpenOnBol_doesNothing(t *testing.T) {
	f := newConfirmFixture()
	f.bol.getOrderFn = openBolOrder(integrationsDomain.BolOrderItem{OrderItemID: "item-1", Quantity: 1, QuantityShipped: 1})

	require.NoError(t, f.run(t))

	assert.Empty(t, f.bol.createShipmentCalls, "already confirmed on Bol (e.g. by hand) -- nothing to send")
	assert.Empty(t, f.confirmations.created)
	assert.Empty(t, f.outbox.Events)
}

func TestConfirmBolShipment_labelOnlyRegistered_waitsForTheCarrier(t *testing.T) {
	f := newConfirmFixture()
	f.shipments[0].StatusMain = "registered"

	require.NoError(t, f.run(t))

	assert.Empty(t, f.source.connectionID, "a label alone may still be cancelled -- Bol is told once the carrier has the parcel")
	assert.Empty(t, f.confirmations.created)
}

func TestConfirmBolShipment_newestLabelNotScannedYet_waitsEvenIfAnOlderOneWas(t *testing.T) {
	f := newConfirmFixture()
	newer := postNLShipment()
	newer.ID, newer.Barcode, newer.StatusMain = "ship-2", "3SSECOND", "registered"
	f.shipments = []shippingDomain.Shipment{newer, postNLShipment()}

	require.NoError(t, f.run(t))

	assert.Empty(t, f.bol.createShipmentCalls)
}

func TestConfirmBolShipment_parcelScannedBeforeOrderIsMarkedShipped_stillSends(t *testing.T) {
	f := newConfirmFixture()
	f.order.Status = omsDomain.SalesOrderStatusProcessing

	require.NoError(t, f.run(t))

	assert.Len(t, f.bol.createShipmentCalls, 1)
}

func TestConfirmBolShipment_orderClosed_doesNothing(t *testing.T) {
	for _, status := range []omsDomain.SalesOrderStatus{omsDomain.SalesOrderStatusCancelled, omsDomain.SalesOrderStatusRefunded, omsDomain.SalesOrderStatusFailed} {
		f := newConfirmFixture()
		f.order.Status = status

		require.NoError(t, f.run(t), status)

		assert.Empty(t, f.source.connectionID, status)
		assert.Empty(t, f.confirmations.created, status)
	}
}

func TestConfirmBolShipment_notABolOrder_doesNothing(t *testing.T) {
	f := newConfirmFixture()
	f.order.Source = omsDomain.SalesOrderSourceWooCommerce

	require.NoError(t, f.run(t))

	assert.Empty(t, f.source.connectionID)
}

func TestConfirmBolShipment_noUsableShipment_doesNothing(t *testing.T) {
	cancelledAt := confirmNow
	cases := map[string][]shippingDomain.Shipment{
		"no shipment":        nil,
		"no barcode yet":     {{ID: "ship-1", Carrier: shippingDomain.CarrierPostNL, StatusMain: "handed_to_carrier"}},
		"shipment cancelled": {{ID: "ship-1", Carrier: shippingDomain.CarrierPostNL, Barcode: "3S1", StatusMain: "handed_to_carrier", CancelledAt: &cancelledAt}},
	}
	for name, shipments := range cases {
		f := newConfirmFixture()
		f.shipments = shipments

		require.NoError(t, f.run(t), name)

		assert.Empty(t, f.bol.createShipmentCalls, name)
		assert.Empty(t, f.confirmations.created, name)
	}
}

func TestConfirmBolShipment_skipsCancelledShipmentForTheActiveOne(t *testing.T) {
	cancelledAt := confirmNow
	f := newConfirmFixture()
	f.shipments = []shippingDomain.Shipment{
		{ID: "ship-new", Carrier: shippingDomain.CarrierDPD, Barcode: "DPD1", RecipientCC: "BE", StatusMain: "handed_to_carrier", CancelledAt: &cancelledAt},
		postNLShipment(),
	}

	require.NoError(t, f.run(t))

	require.Len(t, f.bol.createShipmentCalls, 1)
	assert.Equal(t, "3SMYPA859699114", f.bol.createShipmentCalls[0].TrackAndTrace)
	assert.Equal(t, "ship-1", f.confirmations.created[0].ShipmentID)
}

func TestConfirmBolShipment_alreadySentForThisShipment_doesNotResend(t *testing.T) {
	for _, status := range []channelsDomain.BolConfirmationStatus{channelsDomain.BolConfirmationPending, channelsDomain.BolConfirmationSuccess} {
		f := newConfirmFixture()
		f.confirmations.existing = []channelsDomain.BolShipmentConfirmation{{ID: "conf-old", ShipmentID: "ship-1", Status: status}}

		require.NoError(t, f.run(t), status)

		assert.Empty(t, f.bol.createShipmentCalls, status)
		assert.Empty(t, f.confirmations.created, status)
	}
}

func TestConfirmBolShipment_previousAttemptFailed_sendsAgain(t *testing.T) {
	f := newConfirmFixture()
	f.confirmations.existing = []channelsDomain.BolShipmentConfirmation{{ID: "conf-old", ShipmentID: "ship-1", Status: channelsDomain.BolConfirmationFailure}}

	require.NoError(t, f.run(t))

	assert.Len(t, f.bol.createShipmentCalls, 1)
}

func TestConfirmBolShipment_carrierBolDoesNotKnow_recordsFailureForManualConfirmation(t *testing.T) {
	f := newConfirmFixture()
	f.shipments = []shippingDomain.Shipment{{ID: "ship-1", Carrier: shippingDomain.Carrier(99), Barcode: "X1", StatusMain: "handed_to_carrier"}}

	require.NoError(t, f.run(t))

	assert.Empty(t, f.source.connectionID, "nothing to send to Bol")
	require.Len(t, f.confirmations.created, 1)
	c := f.confirmations.created[0]
	assert.Equal(t, channelsDomain.BolConfirmationFailure, c.Status)
	assert.Contains(t, c.ErrorMessage, "Carrier 99")
	assert.Equal(t, channelsDomain.BolFailureNoTransporterCode, c.FailureReason)
	assert.Empty(t, c.ProcessStatusID)
	event, err := f.outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "failure", event.Data()["status"])
}

func TestConfirmBolShipment_bolRejectsRequest_recordsFailureWithBolsReason(t *testing.T) {
	f := newConfirmFixture()
	f.bol.createShipmentFn = func(context.Context, integrationsDomain.BolShipmentInput) (integrationsDomain.BolProcessStatus, error) {
		return integrationsDomain.BolProcessStatus{}, fmt.Errorf("%w: Order item has already been shipped.", integrationsDomain.ErrBolRequestRejected)
	}

	require.NoError(t, f.run(t), "retrying a rejected request can't help -- record it instead")

	require.Len(t, f.confirmations.created, 1)
	assert.Equal(t, channelsDomain.BolConfirmationFailure, f.confirmations.created[0].Status)
	assert.Contains(t, f.confirmations.created[0].ErrorMessage, "Order item has already been shipped.")
	assert.Equal(t, channelsDomain.BolFailureRejected, f.confirmations.created[0].FailureReason)
	assert.Equal(t, "TNT", f.confirmations.created[0].TransporterCode)
}

func TestConfirmBolShipment_bolUnavailable_returnsErrorForRetry(t *testing.T) {
	f := newConfirmFixture()
	f.bol.createShipmentFn = func(context.Context, integrationsDomain.BolShipmentInput) (integrationsDomain.BolProcessStatus, error) {
		return integrationsDomain.BolProcessStatus{}, errors.New("bol: request failed (503)")
	}

	err := f.run(t)

	require.Error(t, err)
	assert.Empty(t, f.confirmations.created)
}

func TestConfirmBolShipment_cannotReadBolOrder_returnsErrorForRetry(t *testing.T) {
	f := newConfirmFixture()
	f.bol.getOrderFn = func(context.Context, string) (*integrationsDomain.BolOrder, error) {
		return nil, errors.New("bol: timeout")
	}

	require.Error(t, f.run(t))
	assert.Empty(t, f.bol.createShipmentCalls)
}

func TestConfirmBolShipment_cannotBuildClient_returnsError(t *testing.T) {
	f := newConfirmFixture()
	f.source.err = errors.New("decrypt failed")

	require.Error(t, f.run(t))
}

func TestConfirmBolShipment_eventCannotBeRecorded_returnsError(t *testing.T) {
	f := newConfirmFixture()
	f.outbox.Err = errors.New("outbox down")

	require.Error(t, f.run(t))
}

// ── correcting a shipment Bol already has ─────────────────────────────────────

// replacedLabelFixture: Bol was told about ship-old (since cancelled -- the
// wrong label); ship-1 is the replacement the carrier just scanned, and
// nothing is open on Bol any more.
func replacedLabelFixture() *confirmFixture {
	cancelledAt := confirmNow.Add(-time.Hour)
	f := newConfirmFixture()
	f.shipments = []shippingDomain.Shipment{
		postNLShipment(),
		{ID: "ship-old", Carrier: shippingDomain.CarrierPostNL, Barcode: "3SWRONG", StatusMain: "handed_to_carrier", CancelledAt: &cancelledAt},
	}
	f.confirmations.existing = []channelsDomain.BolShipmentConfirmation{{
		ID: "conf-old", ShipmentID: "ship-old", Action: channelsDomain.BolActionCreateShipment,
		TrackAndTrace: "3SWRONG", Status: channelsDomain.BolConfirmationSuccess,
	}}
	f.bol.getOrderFn = openBolOrder(integrationsDomain.BolOrderItem{OrderItemID: "item-1", Quantity: 1, QuantityShipped: 1})
	f.bol.transportIDsFn = func(context.Context, string) ([]string, error) { return []string{"t-1"}, nil }
	return f
}

func TestConfirmBolShipment_replacedLabel_correctsTrackAndTraceOnBol(t *testing.T) {
	f := replacedLabelFixture()

	require.NoError(t, f.run(t))

	assert.Empty(t, f.bol.createShipmentCalls)
	require.Len(t, f.bol.updateTransportCalls, 1)
	assert.Equal(t, fakeUpdateTransportCall{
		TransportID: "t-1",
		Input:       integrationsDomain.BolTransportInput{TransporterCode: "TNT", TrackAndTrace: "3SMYPA859699114"},
	}, f.bol.updateTransportCalls[0])

	require.Len(t, f.confirmations.created, 1)
	c := f.confirmations.created[0]
	assert.Equal(t, channelsDomain.BolActionUpdateTransport, c.Action)
	assert.Equal(t, "t-1", c.TransportID)
	assert.Equal(t, "ship-1", c.ShipmentID)
	assert.Equal(t, "ps-t-1", c.ProcessStatusID)
	assert.Equal(t, channelsDomain.BolConfirmationPending, c.Status)

	event, err := f.outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "update_transport", event.Data()["action"])
	assert.Equal(t, "t-1", event.Data()["transport_id"])
}

func TestConfirmBolShipment_replacedLabel_correctsEveryBolTransportOfTheOrder(t *testing.T) {
	f := replacedLabelFixture()
	f.bol.transportIDsFn = func(context.Context, string) ([]string, error) { return []string{"t-1", "t-2"}, nil }

	require.NoError(t, f.run(t))

	assert.Len(t, f.bol.updateTransportCalls, 2)
	require.Len(t, f.confirmations.created, 2)
	assert.Equal(t, "t-2", f.confirmations.created[1].TransportID)
}

func TestConfirmBolShipment_nothingOpenAndBolNeverToldByUs_leavesBolAlone(t *testing.T) {
	f := replacedLabelFixture()
	f.confirmations.existing = nil

	require.NoError(t, f.run(t))

	assert.Empty(t, f.bol.updateTransportCalls, "confirmed by hand on Bol -- not ours to change")
	assert.Empty(t, f.confirmations.created)
}

func TestConfirmBolShipment_earlierAttemptRejected_isNotCorrected(t *testing.T) {
	f := replacedLabelFixture()
	f.confirmations.existing[0].Status = channelsDomain.BolConfirmationFailure

	require.NoError(t, f.run(t))

	assert.Empty(t, f.bol.updateTransportCalls, "Bol never took the wrong label, nothing to correct")
}

func TestConfirmBolShipment_correctionRejected_recordsFailureWithBolsReason(t *testing.T) {
	f := replacedLabelFixture()
	f.bol.updateTransportFn = func(context.Context, string, integrationsDomain.BolTransportInput) (integrationsDomain.BolProcessStatus, error) {
		return integrationsDomain.BolProcessStatus{}, fmt.Errorf("%w: transport already delivered", integrationsDomain.ErrBolRequestRejected)
	}

	require.NoError(t, f.run(t))

	require.Len(t, f.confirmations.created, 1)
	assert.Equal(t, channelsDomain.BolConfirmationFailure, f.confirmations.created[0].Status)
	assert.Contains(t, f.confirmations.created[0].ErrorMessage, "transport already delivered")
	assert.Equal(t, channelsDomain.BolActionUpdateTransport, f.confirmations.created[0].Action)
}

func TestConfirmBolShipment_correctionButBolHasNoShipment_recordsFailure(t *testing.T) {
	f := replacedLabelFixture()
	f.bol.transportIDsFn = func(context.Context, string) ([]string, error) { return nil, nil }

	require.NoError(t, f.run(t))

	require.Len(t, f.confirmations.created, 1)
	assert.Equal(t, channelsDomain.BolConfirmationFailure, f.confirmations.created[0].Status)
	assert.Equal(t, channelsDomain.BolActionUpdateTransport, f.confirmations.created[0].Action)
	assert.NotEmpty(t, f.confirmations.created[0].ErrorMessage)
	assert.Equal(t, channelsDomain.BolFailureNoBolShipment, f.confirmations.created[0].FailureReason)
}

func TestConfirmBolShipment_cannotListBolTransports_returnsErrorForRetry(t *testing.T) {
	f := replacedLabelFixture()
	f.bol.transportIDsFn = func(context.Context, string) ([]string, error) { return nil, errors.New("bol: 503") }

	require.Error(t, f.run(t))
	assert.Empty(t, f.confirmations.created)
}

func TestConfirmBolShipment_correctionUnavailable_returnsErrorForRetry(t *testing.T) {
	f := replacedLabelFixture()
	f.bol.updateTransportFn = func(context.Context, string, integrationsDomain.BolTransportInput) (integrationsDomain.BolProcessStatus, error) {
		return integrationsDomain.BolProcessStatus{}, errors.New("bol: 503")
	}

	require.Error(t, f.run(t))
}
