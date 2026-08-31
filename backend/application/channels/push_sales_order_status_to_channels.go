package channels

import (
	"context"
	"fmt"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// wooStatusBySalesOrderStatus maps this codebase's internal status
// vocabulary onto WooCommerce's own status strings -- deliberately the same
// set (see domain/oms.SalesOrderStatus's doc, "mirrors WooCommerce's order
// status set exactly") with one exception: SalesOrderStatusShipped only
// ever applies to a non-Woo order (see
// application/oms.HandleOrderStatusWebhookUseCase's Woo-vs-other-source
// branch) and has no Woo equivalent, so it's deliberately absent here.
var wooStatusBySalesOrderStatus = map[omsDomain.SalesOrderStatus]string{
	omsDomain.SalesOrderStatusPending:    "pending",
	omsDomain.SalesOrderStatusProcessing: "processing",
	omsDomain.SalesOrderStatusOnHold:     "on-hold",
	omsDomain.SalesOrderStatusCompleted:  "completed",
	omsDomain.SalesOrderStatusCancelled:  "cancelled",
	omsDomain.SalesOrderStatusRefunded:   "refunded",
	omsDomain.SalesOrderStatusFailed:     "failed",
}

// PushSalesOrderStatusToChannelsUseCase is the consumer-facing half of
// events.SalesOrderStatusChanged (see
// infrastructure/consumers.SalesOrderStatusChangedConsumer) -- "the channel
// decides what to do" for whatever source this order actually came from.
// WooCommerce is the only channel with an outbound status push today;
// pos/manual/bol orders are a deliberate no-op (nothing to push a status
// to yet). Re-fetches the order's current Source/Status rather than
// trusting a caller-supplied snapshot, same reasoning as ProductChanged's
// consumer -- by the time this runs, the order that triggered the event may
// have changed again.
type PushSalesOrderStatusToChannelsUseCase struct {
	db               *gorm.DB
	salesOrderRepo   omsDomain.SalesOrderRepository
	connRepo         integrationsDomain.ConnectionRepository
	wooFactory       integrationsDomain.WooClientFactory
	encryptionSecret string
}

func NewPushSalesOrderStatusToChannelsUseCase(
	db *gorm.DB,
	salesOrderRepo omsDomain.SalesOrderRepository,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
) *PushSalesOrderStatusToChannelsUseCase {
	return &PushSalesOrderStatusToChannelsUseCase{
		db: db, salesOrderRepo: salesOrderRepo, connRepo: connRepo, wooFactory: wooFactory, encryptionSecret: encryptionSecret,
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
		// pos/manual/bol: no outbound status push exists for these channels
		// yet -- each channel's own logic decides, and today that's "do
		// nothing", not an error.
		return nil
	}
}

func (uc *PushSalesOrderStatusToChannelsUseCase) pushToWoo(ctx context.Context, order omsDomain.SalesOrder) error {
	wooStatus, ok := wooStatusBySalesOrderStatus[order.Status]
	if !ok {
		// No Woo equivalent for this status (e.g. "shipped") -- nothing to
		// push, not an error.
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
	return nil
}
