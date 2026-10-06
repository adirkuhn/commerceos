package channels

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// SetSalesChannelPaymentMethodUseCase is the dedicated "enable/disable this
// channel's payment method" action -- deliberately separate from
// UpdateSalesChannelUseCase's general edit form, same granularity
// convention as SetProductChannelOverrideUseCase. paymentConnectionID nil
// disables payment on this channel (SalesChannel.PaymentConnectionID's own
// doc comment); non-nil must reference a connection this tenant actually
// has connected and that is a payment integration -- re-validated here
// server-side rather than trusting whatever the payment-method picker UI
// last fetched from ListPaymentConnectionsUseCase.
type SetSalesChannelPaymentMethodUseCase struct {
	db       *gorm.DB
	repo     channelsDomain.SalesChannelRepository
	connRepo integrationsDomain.ConnectionRepository
	outbox   eventrecord.Enqueuer
}

func NewSetSalesChannelPaymentMethodUseCase(db *gorm.DB, repo channelsDomain.SalesChannelRepository, connRepo integrationsDomain.ConnectionRepository, outbox eventrecord.Enqueuer) *SetSalesChannelPaymentMethodUseCase {
	return &SetSalesChannelPaymentMethodUseCase{db: db, repo: repo, connRepo: connRepo, outbox: outbox}
}

func (uc *SetSalesChannelPaymentMethodUseCase) Execute(ctx context.Context, channelID string, paymentConnectionID *string) (*channelsDomain.SalesChannel, error) {
	var channel channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, channelID)
		if err != nil {
			return err
		}
		if paymentConnectionID != nil {
			conn, err := uc.connRepo.FindByID(ctx, *paymentConnectionID)
			if err != nil {
				if errors.Is(err, integrationsDomain.ErrConnectionNotFound) {
					return channelsDomain.ErrInvalidPaymentConnection
				}
				return err
			}
			if !conn.Provider.IsPaymentProvider() || conn.Status != integrationsDomain.StatusConnected {
				return channelsDomain.ErrInvalidPaymentConnection
			}
		}
		before := *existing
		existing.PaymentConnectionID = paymentConnectionID
		existing.UpdatedAt = time.Now()
		if err := uc.repo.Update(ctx, existing); err != nil {
			return err
		}
		channel = *existing
		return eventrecord.Record(ctx, uc.outbox, events.SalesChannelUpdated, channel.ID, salesChannelChanges(before, channel))
	})
	if err != nil {
		return nil, fmt.Errorf("setSalesChannelPaymentMethod: %w", err)
	}
	return &channel, nil
}
