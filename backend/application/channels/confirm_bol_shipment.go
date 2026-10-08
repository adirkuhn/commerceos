package channels

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// closedOrderStatuses: a Bol order in one of these is never confirmed on Bol.
var closedOrderStatuses = map[omsDomain.SalesOrderStatus]bool{
	omsDomain.SalesOrderStatusCancelled: true,
	omsDomain.SalesOrderStatusRefunded:  true,
	omsDomain.SalesOrderStatusFailed:    true,
}

// ConfirmBolShipmentUseCase tells Bol that a Bol order's parcel left, so the
// order closes on Bol's side too. Runs on every shipment status update (see
// infrastructure/consumers.BolShipmentConfirmationConsumer) and acts only
// once the order's newest active shipment is in the carrier's hands
// (shippingDomain.IsCarrierConfirmed) -- never on a label alone, which may
// still be cancelled as a mistake.
//
// Then, for the items still open on Bol, it sends the shipment
// (POST /retailer/shipments, transporter code + track & trace). When nothing
// is open on Bol but CommerceOS told Bol about a different, since cancelled
// shipment, the parcel was replaced: it corrects the transporter + track &
// trace of the order's shipments on Bol (PUT /retailer/transports/{id}).
// An order nothing was sent for (e.g. confirmed by hand on Bol) is left alone.
//
// Every Bol request is stored as a BolShipmentConfirmation so the order page
// can show what happened: pending while Bol processes it (resolved later by
// CheckBolShipmentConfirmationsUseCase), or failed right away with the
// reason when it can't be sent -- Bol refused the request, or the carrier
// has no Bol transporter code. Failures a retry could fix (Bol or the
// network unavailable) are returned instead, so the event is redelivered.
// Never sends twice for a shipment Bol already has (pending or success).
type ConfirmBolShipmentUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	shipmentRepo   shippingDomain.ShipmentRepository
	confirmations  channelsDomain.BolShipmentConfirmationRepository
	clients        bolClientSource
	outbox         eventrecord.Enqueuer
	now            func() time.Time
}

func NewConfirmBolShipmentUseCase(
	db *gorm.DB,
	salesOrderRepo omsDomain.SalesOrderRepository,
	shipmentRepo shippingDomain.ShipmentRepository,
	confirmations channelsDomain.BolShipmentConfirmationRepository,
	clients bolClientSource,
	outbox eventrecord.Enqueuer,
	now func() time.Time,
) *ConfirmBolShipmentUseCase {
	return &ConfirmBolShipmentUseCase{
		db: db, salesOrderRepo: salesOrderRepo, shipmentRepo: shipmentRepo, confirmations: confirmations,
		clients: clients, outbox: outbox, now: now,
	}
}

func (uc *ConfirmBolShipmentUseCase) Execute(ctx context.Context, salesOrderID string) error {
	order, err := uc.loadOrder(ctx, salesOrderID)
	if err != nil {
		return fmt.Errorf("confirmBolShipment: %w", err)
	}
	if order.Source != omsDomain.SalesOrderSourceBol || closedOrderStatuses[order.Status] || order.ConnectionID == nil {
		return nil
	}

	shipments, confirmations, err := uc.loadShipments(ctx, order.ID)
	if err != nil {
		return fmt.Errorf("confirmBolShipment: %w", err)
	}
	shipment := newestActiveShipment(shipments)
	if shipment == nil {
		log.Printf("confirmBolShipment: order %s has no active shipment with a barcode -- nothing to send to Bol", order.ID)
		return nil
	}
	if !shippingDomain.IsCarrierConfirmed(shipment.StatusMain) || toldBolAbout(confirmations, shipment.ID) {
		return nil
	}

	base := channelsDomain.BolShipmentConfirmation{
		TenantID: order.TenantID, SalesOrderID: order.ID, ConnectionID: *order.ConnectionID, ShipmentID: shipment.ID,
		ExternalOrderID: order.ExternalOrderID, Action: channelsDomain.BolActionCreateShipment,
		TrackAndTrace: shipment.Barcode, CreatedAt: uc.now(),
	}
	transporterCode, err := channelsDomain.BolTransporterCode(shipment.Carrier, shipment.RecipientCC)
	if err != nil {
		return uc.save(ctx, failed(base, fmt.Sprintf("Bol has no transporter code for %s -- confirm this order in Bol yourself.", shipment.Carrier)))
	}
	base.TransporterCode = transporterCode

	client, err := uc.clients.ClientFor(ctx, *order.ConnectionID)
	if err != nil {
		return fmt.Errorf("confirmBolShipment: %w", err)
	}
	bolOrder, err := client.GetOrder(ctx, order.ExternalOrderID)
	if err != nil {
		return fmt.Errorf("confirmBolShipment: read bol order %s: %w", order.ExternalOrderID, err)
	}
	if items := openBolOrderItems(bolOrder.OrderItems); len(items) > 0 {
		return uc.createShipment(ctx, client, base, items)
	}
	if toldBolAboutAnother(confirmations, shipment.ID) {
		return uc.correctTransports(ctx, client, base)
	}
	return nil
}

func (uc *ConfirmBolShipmentUseCase) createShipment(ctx context.Context, client integrationsDomain.BolClient, c channelsDomain.BolShipmentConfirmation, items []integrationsDomain.BolShipmentItem) error {
	status, err := client.CreateShipment(ctx, integrationsDomain.BolShipmentInput{
		OrderItems: items, ShipmentReference: c.ExternalOrderID,
		TransporterCode: c.TransporterCode, TrackAndTrace: c.TrackAndTrace,
	})
	if errors.Is(err, integrationsDomain.ErrBolRequestRejected) {
		return uc.save(ctx, failed(c, err.Error()))
	}
	if err != nil {
		return fmt.Errorf("confirmBolShipment: send shipment for bol order %s: %w", c.ExternalOrderID, err)
	}
	accepted, err := uc.accepted(c, status)
	if err != nil {
		return err
	}
	return uc.save(ctx, accepted)
}

// correctTransports replaces the transporter + track & trace on every
// shipment Bol has for the order. All requests are made before anything is
// stored: a retryable failure returns an error with nothing saved, and the
// retry repeats the (idempotent) updates.
func (uc *ConfirmBolShipmentUseCase) correctTransports(ctx context.Context, client integrationsDomain.BolClient, base channelsDomain.BolShipmentConfirmation) error {
	base.Action = channelsDomain.BolActionUpdateTransport
	transportIDs, err := client.ListShipmentTransportIDs(ctx, base.ExternalOrderID)
	if err != nil {
		return fmt.Errorf("confirmBolShipment: list bol shipments for order %s: %w", base.ExternalOrderID, err)
	}
	if len(transportIDs) == 0 {
		return uc.save(ctx, failed(base, fmt.Sprintf("Bol has no shipment for order %s to correct -- update the track & trace in Bol yourself.", base.ExternalOrderID)))
	}

	results := make([]channelsDomain.BolShipmentConfirmation, 0, len(transportIDs))
	for _, transportID := range transportIDs {
		c := base
		c.TransportID = transportID
		status, err := client.UpdateTransport(ctx, transportID, integrationsDomain.BolTransportInput{
			TransporterCode: c.TransporterCode, TrackAndTrace: c.TrackAndTrace,
		})
		if errors.Is(err, integrationsDomain.ErrBolRequestRejected) {
			results = append(results, failed(c, err.Error()))
			continue
		}
		if err != nil {
			return fmt.Errorf("confirmBolShipment: correct bol transport %s: %w", transportID, err)
		}
		accepted, err := uc.accepted(c, status)
		if err != nil {
			return err
		}
		results = append(results, accepted)
	}
	return uc.save(ctx, results...)
}

// accepted records Bol's process status; Bol may already have a final answer.
func (uc *ConfirmBolShipmentUseCase) accepted(c channelsDomain.BolShipmentConfirmation, status integrationsDomain.BolProcessStatus) (channelsDomain.BolShipmentConfirmation, error) {
	c.ProcessStatusID = status.ID
	c.Status = channelsDomain.BolConfirmationPending
	if final, err := channelsDomain.ParseBolConfirmationStatus(status.Status); err == nil && final != channelsDomain.BolConfirmationPending {
		if err := c.Resolve(final, status.ErrorMessage, uc.now()); err != nil {
			return c, fmt.Errorf("confirmBolShipment: %w", err)
		}
	}
	return c, nil
}

func failed(c channelsDomain.BolShipmentConfirmation, reason string) channelsDomain.BolShipmentConfirmation {
	c.Status = channelsDomain.BolConfirmationFailure
	c.ErrorMessage = reason
	return c
}

func (uc *ConfirmBolShipmentUseCase) loadOrder(ctx context.Context, salesOrderID string) (*omsDomain.SalesOrder, error) {
	var order *omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		order, err = uc.salesOrderRepo.FindByID(ctx, salesOrderID)
		return err
	})
	return order, err
}

func (uc *ConfirmBolShipmentUseCase) loadShipments(ctx context.Context, salesOrderID string) ([]shippingDomain.Shipment, []channelsDomain.BolShipmentConfirmation, error) {
	var shipments []shippingDomain.Shipment
	var confirmations []channelsDomain.BolShipmentConfirmation
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		if shipments, err = uc.shipmentRepo.FindBySalesOrderID(ctx, salesOrderID); err != nil {
			return err
		}
		confirmations, err = uc.confirmations.ListBySalesOrder(ctx, salesOrderID)
		return err
	})
	return shipments, confirmations, err
}

// newestActiveShipment: shipments come newest first; the first one not
// cancelled and with a barcode is the parcel that counts.
func newestActiveShipment(shipments []shippingDomain.Shipment) *shippingDomain.Shipment {
	for i := range shipments {
		if shipments[i].CancelledAt == nil && shipments[i].Barcode != "" {
			return &shipments[i]
		}
	}
	return nil
}

func toldBolAbout(confirmations []channelsDomain.BolShipmentConfirmation, shipmentID string) bool {
	for _, c := range confirmations {
		if c.ShipmentID == shipmentID && c.ToldBol() {
			return true
		}
	}
	return false
}

func toldBolAboutAnother(confirmations []channelsDomain.BolShipmentConfirmation, shipmentID string) bool {
	for _, c := range confirmations {
		if c.ShipmentID != shipmentID && c.ToldBol() {
			return true
		}
	}
	return false
}

// openBolOrderItems lists what Bol still considers unshipped: per item,
// ordered minus shipped minus cancelled.
func openBolOrderItems(items []integrationsDomain.BolOrderItem) []integrationsDomain.BolShipmentItem {
	var open []integrationsDomain.BolShipmentItem
	for _, it := range items {
		if qty := it.Quantity - it.QuantityShipped - it.QuantityCancelled; qty > 0 {
			open = append(open, integrationsDomain.BolShipmentItem{OrderItemID: it.OrderItemID, Quantity: qty})
		}
	}
	return open
}

func (uc *ConfirmBolShipmentUseCase) save(ctx context.Context, confirmations ...channelsDomain.BolShipmentConfirmation) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		for _, c := range confirmations {
			if err := uc.confirmations.Create(ctx, &c); err != nil {
				return err
			}
			if err := eventrecord.Record(ctx, uc.outbox, events.BolShipmentConfirmationSent, c.ID, eventrecord.Data{
				"sales_order_id": c.SalesOrderID, "shipment_id": c.ShipmentID, "connection_id": c.ConnectionID,
				"external_order_id": c.ExternalOrderID, "action": string(c.Action), "transport_id": c.TransportID,
				"transporter_code": c.TransporterCode, "track_and_trace": c.TrackAndTrace,
				"process_status_id": c.ProcessStatusID, "status": string(c.Status), "error_message": c.ErrorMessage,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("confirmBolShipment: save confirmation: %w", err)
	}
	return nil
}
