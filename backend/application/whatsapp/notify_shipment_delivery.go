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

// collectPointStatusCodes are the MyParcel numeric shipment status codes
// that mean "delivered, but left at a collect/pickup point" rather than
// handed directly to the recipient -- 8 ("ready for pickup") and 19 ("at
// agreed location"). See shipping.MyParcelStatusLabel: every code in the
// 7/8/9/19 family reports status_main "delivered", so distinguishing these
// needs the raw numeric code, not status_main. Package-level (not just a
// local var) since it's the one place this distinction is made -- keep it
// here if MyParcel's code set is ever revisited.
var collectPointStatusCodes = map[int]bool{8: true, 19: true}

// NotifyShipmentDeliveryUseCase is application/shipping's
// ShipmentDeliveryNotifier -- called (see apply_status_update.go's
// notifyWhatsAppOnDelivery) at the same trigger point as
// completeOrderOnDelivery, but needs the shipment event's raw status code
// to pick the right wording. Same customer/phone/channel resolution and
// "silently skip if nothing to notify" rule as NotifyOrderStatusUseCase.
type NotifyShipmentDeliveryUseCase struct {
	db               *gorm.DB
	salesOrderRepo   omsDomain.SalesOrderRepository
	customerRepo     customersDomain.CustomerRepository
	connRepo         integrationsDomain.ConnectionRepository
	evolutionFactory integrationsDomain.EvolutionAPIClientFactory
	encryptionSecret string
}

func NewNotifyShipmentDeliveryUseCase(
	db *gorm.DB,
	salesOrderRepo omsDomain.SalesOrderRepository,
	customerRepo customersDomain.CustomerRepository,
	connRepo integrationsDomain.ConnectionRepository,
	evolutionFactory integrationsDomain.EvolutionAPIClientFactory,
	encryptionSecret string,
) *NotifyShipmentDeliveryUseCase {
	return &NotifyShipmentDeliveryUseCase{
		db: db, salesOrderRepo: salesOrderRepo, customerRepo: customerRepo, connRepo: connRepo,
		evolutionFactory: evolutionFactory, encryptionSecret: encryptionSecret,
	}
}

func (uc *NotifyShipmentDeliveryUseCase) Execute(ctx context.Context, salesOrderID string, statusCode int) error {
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
		return fmt.Errorf("notifyShipmentDelivery: %w", err)
	}
	if customer == nil || conn == nil {
		return nil
	}

	client, err := buildEvolutionClient(uc.evolutionFactory, uc.encryptionSecret, *conn)
	if err != nil {
		return fmt.Errorf("notifyShipmentDelivery: %w", err)
	}

	message := fmt.Sprintf("Hi %s! Your order #%s: %s", customer.FirstName, order.Number, deliveryMessage(statusCode))
	if err := client.SendText(ctx, customer.PhoneE164, message); err != nil {
		return fmt.Errorf("notifyShipmentDelivery: send: %w", err)
	}
	return nil
}

func deliveryMessage(statusCode int) string {
	if collectPointStatusCodes[statusCode] {
		return "your order was delivered to a collect point near you -- please pick it up."
	}
	return "your order was delivered!"
}
