package channels

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"log"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// wooStatusBySalesOrderStatus maps this codebase's internal status
// vocabulary onto WooCommerce's own status strings -- deliberately the same
// set (see domain/oms.SalesOrderStatus's doc, "mirrors WooCommerce's order
// status set exactly") with one exception: Shipped has no native Woo
// equivalent, so it maps onto Woo's own "on-hold" -- CommerceOS's internal
// record still says Shipped (see
// application/shipping.MarkOrderShippedOnFulfillment, which never special-
// cases Woo), only the outbound push translates it, and pushToWoo attaches
// a shipment-info order note alongside it (see addShipmentNote below) so
// Woo staff can tell this on-hold apart from any other reason an order
// lands there.
var wooStatusBySalesOrderStatus = map[omsDomain.SalesOrderStatus]string{
	omsDomain.SalesOrderStatusPending:    "pending",
	omsDomain.SalesOrderStatusProcessing: "processing",
	omsDomain.SalesOrderStatusOnHold:     "on-hold",
	omsDomain.SalesOrderStatusShipped:    "on-hold",
	omsDomain.SalesOrderStatusCompleted:  "completed",
	omsDomain.SalesOrderStatusCancelled:  "cancelled",
	omsDomain.SalesOrderStatusRefunded:   "refunded",
	omsDomain.SalesOrderStatusFailed:     "failed",
}

// PushSalesOrderStatusToChannelsUseCase is the consumer-facing half of
// events.SalesOrderStatusChanged (see
// infrastructure/consumers.SalesOrderStatusChangedConsumer) -- "the channel
// decides what to do" for whatever source this order actually came from.
// WooCommerce is the only channel with an outbound status push; pos/manual
// orders have nowhere to push to, and Bol orders are confirmed on Bol by
// ConfirmBolShipmentUseCase once the carrier scans the parcel. Re-fetches the order's current Source/Status rather than
// trusting a caller-supplied snapshot, same reasoning as ProductChanged's
// consumer -- by the time this runs, the order that triggered the event may
// have changed again.
type PushSalesOrderStatusToChannelsUseCase struct {
	db                  *gorm.DB
	salesOrderRepo      omsDomain.SalesOrderRepository
	shipmentRepo        shippingDomain.ShipmentRepository
	carrierOverrideRepo shippingDomain.ShippingCarrierOverrideRepository
	connRepo            integrationsDomain.ConnectionRepository
	wooFactory          integrationsDomain.WooClientFactory
	encryptionSecret    string
	outbox              eventrecord.Enqueuer
}

func NewPushSalesOrderStatusToChannelsUseCase(
	db *gorm.DB,
	salesOrderRepo omsDomain.SalesOrderRepository,
	shipmentRepo shippingDomain.ShipmentRepository,
	carrierOverrideRepo shippingDomain.ShippingCarrierOverrideRepository,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
	outbox eventrecord.Enqueuer,
) *PushSalesOrderStatusToChannelsUseCase {
	return &PushSalesOrderStatusToChannelsUseCase{
		db: db, salesOrderRepo: salesOrderRepo, shipmentRepo: shipmentRepo, carrierOverrideRepo: carrierOverrideRepo,
		connRepo: connRepo, wooFactory: wooFactory, encryptionSecret: encryptionSecret,
		outbox: outbox,
	}
}

func (uc *PushSalesOrderStatusToChannelsUseCase) Execute(ctx context.Context, salesOrderID string) error {
	var order omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		o, err := uc.salesOrderRepo.FindByID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		order = *o
		return nil
	})
	if err != nil {
		return fmt.Errorf("pushSalesOrderStatusToChannels: %w", err)
	}

	switch order.Source {
	case omsDomain.SalesOrderSourceWooCommerce:
		return uc.pushToWoo(ctx, order)
	default:
		// pos/manual: nothing to push to. bol: Bol has no order status to
		// set -- a Bol order is confirmed on Bol once the carrier scans
		// its parcel (ConfirmBolShipmentUseCase, on the shipment signal).
		return nil
	}
}

func (uc *PushSalesOrderStatusToChannelsUseCase) pushToWoo(ctx context.Context, order omsDomain.SalesOrder) error {
	wooStatus, ok := wooStatusBySalesOrderStatus[order.Status]
	if !ok {
		// No Woo equivalent for this status -- every current SalesOrderStatus
		// value is mapped above, so this is defensive against a future
		// status being added here before this map learns about it; nothing
		// to push, not an error.
		return nil
	}
	if order.ConnectionID == nil {
		// Shouldn't happen for a Woo-sourced order (ImportWooOrdersUseCase
		// always sets it), but nothing to push to without one.
		return nil
	}

	var conn integrationsDomain.Connection
	var secret string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, *order.ConnectionID)
		if err != nil {
			return err
		}
		conn = *c
		secret, err = crypto.Decrypt(uc.encryptionSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return fmt.Errorf("pushSalesOrderStatusToChannels: %w", err)
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return fmt.Errorf("pushSalesOrderStatusToChannels: build client: %w", err)
	}
	if err := client.UpdateOrderStatus(ctx, order.ExternalOrderID, wooStatus); err != nil {
		return fmt.Errorf("pushSalesOrderStatusToChannels: %w", err)
	}
	if err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return eventrecord.Record(ctx, uc.outbox, events.ChannelOrderStatusPushed, order.ID, eventrecord.Data{
			"provider": string(integrationsDomain.ProviderWooCommerce), "connection_id": order.ConnectionID,
			"external_order_id": order.ExternalOrderID, "status": string(order.Status), "channel_status": wooStatus,
		})
	}); err != nil {
		return fmt.Errorf("pushSalesOrderStatusToChannels: %w", err)
	}

	if order.Status == omsDomain.SalesOrderStatusShipped {
		uc.addShipmentNote(ctx, client, order)
	}
	return nil
}

// addShipmentNote attaches the order's shipment info (carrier, tracking) as
// a Woo order note -- Shipped is the internal status
// MarkOrderShippedOnFulfillment sets specifically on fulfillment, so unlike
// OnHold (which staff can also set by hand, for an unrelated reason like a
// payment review) it's an unambiguous "this on-hold push is the shipped
// one" signal, no separate Fulfilled check needed. Best-effort: a failure
// here is logged, not returned -- the status push itself already succeeded
// and is the part that must not be retried into a duplicate Woo call; the
// note is a courtesy on top of it.
func (uc *PushSalesOrderStatusToChannelsUseCase) addShipmentNote(ctx context.Context, client integrationsDomain.WooClient, order omsDomain.SalesOrder) {
	var shipments []shippingDomain.Shipment
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		shipments, err = uc.shipmentRepo.FindBySalesOrderID(ctx, order.ID)
		return err
	})
	if err != nil {
		log.Printf("pushSalesOrderStatusToChannels: shipment note: list shipments for order %s: %v", order.ID, err)
		return
	}
	if len(shipments) == 0 {
		return
	}
	// FindBySalesOrderID returns newest first -- the most recent shipment is
	// the one that actually triggered this fulfillment.
	s := shipments[0]
	note := fmt.Sprintf("Shipped via %s, tracking %s", s.Carrier, s.Barcode)
	if link := uc.resolveTrackingLink(ctx, s); link != "" {
		note += fmt.Sprintf(" - %s", link)
	}
	if err := client.AddOrderNote(ctx, order.ExternalOrderID, note); err != nil {
		log.Printf("pushSalesOrderStatusToChannels: shipment note: order %s: %v", order.ID, err)
	}
}

// resolveTrackingLink prefers the connection's configured carrier tracking
// URL template (see domain/shipping.ShippingCarrierOverride.TrackingURLTemplate's
// doc) over the shipment's own TrackingURL (MyParcel's link_consumer_portal
// -- not always present or correct for every carrier). Best-effort: a
// lookup failure just falls back to the shipment's own value, same as "no
// override configured."
func (uc *PushSalesOrderStatusToChannelsUseCase) resolveTrackingLink(ctx context.Context, s shippingDomain.Shipment) string {
	var override *shippingDomain.ShippingCarrierOverride
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		override, err = uc.carrierOverrideRepo.FindByConnectionAndCarrier(ctx, s.ConnectionID, s.Carrier)
		return err
	})
	if err != nil {
		log.Printf("pushSalesOrderStatusToChannels: shipment note: resolve carrier override for shipment %s: %v", s.ID, err)
		return s.TrackingURL
	}
	if override == nil {
		return s.TrackingURL
	}
	if link := shippingDomain.BuildCarrierTrackingURL(override.TrackingURLTemplate, s.Barcode, s.RecipientPostalCode); link != "" {
		return link
	}
	return s.TrackingURL
}
