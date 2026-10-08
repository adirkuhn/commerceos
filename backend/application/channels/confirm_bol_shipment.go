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

// ConfirmBolShipmentUseCase tells Bol that a Bol order shipped, so the
// order closes on Bol's side too (POST /retailer/shipments with the
// parcel's transporter code and track & trace). Runs on
// oms.sales_order.status_changed (see
// infrastructure/consumers.BolShipmentConfirmationConsumer) and only acts
// on a Bol order that is now Shipped.
//
// Every attempt is stored as a BolShipmentConfirmation so the order page
// can show what happened: pending while Bol processes it (resolved later
// by CheckBolShipmentConfirmationsUseCase), or failed right away when it
// can't be sent -- Bol refused the request, or the carrier has no Bol
// transporter code. Failures that a retry could fix (Bol or the network
// unavailable) are returned instead, so the event is redelivered.
//
// Safe to run again for the same order: it sends only what is still open
// on Bol, and never re-sends a shipment that already has a pending or
// successful confirmation.
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
	if order.Source != omsDomain.SalesOrderSourceBol || order.Status != omsDomain.SalesOrderStatusShipped || order.ConnectionID == nil {
		return nil
	}

	shipment, alreadySent, err := uc.findShipmentToConfirm(ctx, order.ID)
	if err != nil {
		return fmt.Errorf("confirmBolShipment: %w", err)
	}
	if shipment == nil {
		log.Printf("confirmBolShipment: order %s is shipped but has no active shipment with a barcode -- nothing to send to Bol", order.ID)
		return nil
	}
	if alreadySent {
		return nil
	}

	confirmation := channelsDomain.BolShipmentConfirmation{
		TenantID: order.TenantID, SalesOrderID: order.ID, ConnectionID: *order.ConnectionID, ShipmentID: shipment.ID,
		ExternalOrderID: order.ExternalOrderID, TrackAndTrace: shipment.Barcode, CreatedAt: uc.now(),
	}

	transporterCode, err := channelsDomain.BolTransporterCode(shipment.Carrier, shipment.RecipientCC)
	if err != nil {
		confirmation.Status = channelsDomain.BolConfirmationFailure
		confirmation.ErrorMessage = fmt.Sprintf("Bol has no transporter code for %s -- confirm this order in Bol yourself.", shipment.Carrier)
		return uc.save(ctx, confirmation)
	}
	confirmation.TransporterCode = transporterCode

	client, err := uc.clients.ClientFor(ctx, *order.ConnectionID)
	if err != nil {
		return fmt.Errorf("confirmBolShipment: %w", err)
	}
	bolOrder, err := client.GetOrder(ctx, order.ExternalOrderID)
	if err != nil {
		return fmt.Errorf("confirmBolShipment: read bol order %s: %w", order.ExternalOrderID, err)
	}
	items := openBolOrderItems(bolOrder.OrderItems)
	if len(items) == 0 {
		return nil
	}

	status, err := client.CreateShipment(ctx, integrationsDomain.BolShipmentInput{
		OrderItems: items, ShipmentReference: order.ExternalOrderID,
		TransporterCode: transporterCode, TrackAndTrace: shipment.Barcode,
	})
	if errors.Is(err, integrationsDomain.ErrBolRequestRejected) {
		confirmation.Status = channelsDomain.BolConfirmationFailure
		confirmation.ErrorMessage = err.Error()
		return uc.save(ctx, confirmation)
	}
	if err != nil {
		return fmt.Errorf("confirmBolShipment: send shipment for bol order %s: %w", order.ExternalOrderID, err)
	}

	confirmation.ProcessStatusID = status.ID
	confirmation.Status = channelsDomain.BolConfirmationPending
	if final, err := channelsDomain.ParseBolConfirmationStatus(status.Status); err == nil && final != channelsDomain.BolConfirmationPending {
		if err := confirmation.Resolve(final, status.ErrorMessage, uc.now()); err != nil {
			return fmt.Errorf("confirmBolShipment: %w", err)
		}
	}
	return uc.save(ctx, confirmation)
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

// findShipmentToConfirm picks the order's newest active shipment that has
// a barcode, and reports whether it was already sent to Bol (a pending or
// successful confirmation exists for it).
func (uc *ConfirmBolShipmentUseCase) findShipmentToConfirm(ctx context.Context, salesOrderID string) (*shippingDomain.Shipment, bool, error) {
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
	if err != nil {
		return nil, false, err
	}

	for i := range shipments {
		s := shipments[i]
		if s.CancelledAt != nil || s.Barcode == "" {
			continue
		}
		for _, c := range confirmations {
			if c.ShipmentID == s.ID && !c.NeedsManualAction() {
				return &s, true, nil
			}
		}
		return &s, false, nil
	}
	return nil, false, nil
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

func (uc *ConfirmBolShipmentUseCase) save(ctx context.Context, c channelsDomain.BolShipmentConfirmation) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.confirmations.Create(ctx, &c); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.BolShipmentConfirmationSent, c.ID, eventrecord.Data{
			"sales_order_id": c.SalesOrderID, "shipment_id": c.ShipmentID, "connection_id": c.ConnectionID,
			"external_order_id": c.ExternalOrderID, "transporter_code": c.TransporterCode,
			"track_and_trace": c.TrackAndTrace, "process_status_id": c.ProcessStatusID,
			"status": string(c.Status), "error_message": c.ErrorMessage,
		})
	})
	if err != nil {
		return fmt.Errorf("confirmBolShipment: save confirmation: %w", err)
	}
	return nil
}
