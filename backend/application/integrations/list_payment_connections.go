package integrations

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListPaymentConnectionsUseCase lists the tenant's connected payment
// integrations (Stripe/Revolut, see Provider.IsPaymentProvider) -- powers
// the WhatsApp channel's payment-method picker (see
// application/channels.SetSalesChannelPaymentMethodUseCase, which
// re-validates against the same filter server-side rather than trusting
// this list alone).
type ListPaymentConnectionsUseCase struct {
	db       *gorm.DB
	connRepo integrationsDomain.ConnectionRepository
}

func NewListPaymentConnectionsUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository) *ListPaymentConnectionsUseCase {
	return &ListPaymentConnectionsUseCase{db: db, connRepo: connRepo}
}

func (uc *ListPaymentConnectionsUseCase) Execute(ctx context.Context) ([]integrationsDomain.Connection, error) {
	var result []integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		conns, err := uc.connRepo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		for _, c := range conns {
			if c.Provider.IsPaymentProvider() && c.Status == integrationsDomain.StatusConnected {
				result = append(result, c)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listPaymentConnections: %w", err)
	}
	return result, nil
}
