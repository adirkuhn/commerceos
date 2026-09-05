package whatsapp

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// NotifyOrderStatusUseCase is the WhatsApp consumer for
// oms.sales_order.status_changed (see infrastructure/consumers.WhatsAppOrderStatusConsumer)
// -- sends a status-update message to the order's customer, but ONLY when
// that customer is already resolvable by an E.164 phone (Customer.PhoneE164):
// a guest/walk-in sale (CustomerID nil) or a linked customer with no phone
// on file gets no message, since there is nowhere verified to send it.
// SalesOrder.Billing.Phone is deliberately not used as a fallback -- it's
// free-text (WooCommerce-shaped), not guaranteed E.164, and guessing wrong
// risks messaging a stranger's number.
type NotifyOrderStatusUseCase struct {
	db               *gorm.DB
	salesOrderRepo   omsDomain.SalesOrderRepository
	customerRepo     customersDomain.CustomerRepository
	connRepo         integrationsDomain.ConnectionRepository
	evolutionFactory integrationsDomain.EvolutionAPIClientFactory
	encryptionSecret string
}

func NewNotifyOrderStatusUseCase(
	db *gorm.DB,
	salesOrderRepo omsDomain.SalesOrderRepository,
	customerRepo customersDomain.CustomerRepository,
	connRepo integrationsDomain.ConnectionRepository,
	evolutionFactory integrationsDomain.EvolutionAPIClientFactory,
	encryptionSecret string,
) *NotifyOrderStatusUseCase {
	return &NotifyOrderStatusUseCase{
		db: db, salesOrderRepo: salesOrderRepo, customerRepo: customerRepo, connRepo: connRepo,
		evolutionFactory: evolutionFactory, encryptionSecret: encryptionSecret,
	}
}

func (uc *NotifyOrderStatusUseCase) Execute(ctx context.Context, salesOrderID string) error {
	var order *omsDomain.SalesOrder
	var customer *customersDomain.Customer
	var conn *integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		o, err := uc.salesOrderRepo.FindByID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		order = o
		if order.CustomerID == nil {
			return nil
		}
		c, err := uc.customerRepo.FindByID(ctx, *order.CustomerID)
		if err != nil {
			return err
		}
		if c.PhoneE164 == "" {
			return nil
		}
		customer = c

		conns, err := uc.connRepo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		conn = findWhatsAppConnection(conns)
		return nil
	})
	if err != nil {
		return fmt.Errorf("notifyOrderStatus: %w", err)
	}
	if customer == nil || conn == nil {
		return nil
	}

	client, err := buildEvolutionClient(uc.evolutionFactory, uc.encryptionSecret, *conn)
	if err != nil {
		return fmt.Errorf("notifyOrderStatus: %w", err)
	}

	message := fmt.Sprintf("Hi %s! Your order #%s: %s", customer.FirstName, order.Number, orderStatusMessage(order.Status))
	if err := client.SendText(ctx, customer.PhoneE164, message); err != nil {
		return fmt.Errorf("notifyOrderStatus: send: %w", err)
	}
	return nil
}

// orderStatusMessage renders SalesOrderStatus as customer-facing text --
// deliberately not just the raw status string (e.g. "on-hold" means nothing
// to a customer).
func orderStatusMessage(status omsDomain.SalesOrderStatus) string {
	switch status {
	case omsDomain.SalesOrderStatusPending:
		return "we've received your order."
	case omsDomain.SalesOrderStatusProcessing:
		return "we're preparing your order."
	case omsDomain.SalesOrderStatusOnHold:
		return "your order is on hold."
	case omsDomain.SalesOrderStatusShipped:
		return "your order is on its way!"
	case omsDomain.SalesOrderStatusCompleted:
		return "your order was delivered. Thank you!"
	case omsDomain.SalesOrderStatusCancelled:
		return "your order was cancelled."
	case omsDomain.SalesOrderStatusRefunded:
		return "your order was refunded."
	case omsDomain.SalesOrderStatusFailed:
		return "there was a problem with your order."
	default:
		return "your order status changed to " + string(status) + "."
	}
}
