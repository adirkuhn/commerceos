package notifications

import (
	"context"
	"fmt"

	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
)

// notificationDeliverer is DeliverNotificationUseCase narrowed to point of use.
type notificationDeliverer interface {
	Execute(ctx context.Context, in CreateNotificationInput) (*notificationsDomain.Notification, error)
}

// NotifyStockCloseToExpiryUseCase puts one stock-close-to-expiry report in
// the bell (per the stock category's in-app toggle). Its email is the
// Message Center rule's job -- see TypeStockCloseToExpiry.
type NotifyStockCloseToExpiryUseCase struct {
	deliver notificationDeliverer
}

func NewNotifyStockCloseToExpiryUseCase(deliver notificationDeliverer) *NotifyStockCloseToExpiryUseCase {
	return &NotifyStockCloseToExpiryUseCase{deliver: deliver}
}

func (uc *NotifyStockCloseToExpiryUseCase) Execute(ctx context.Context, windowDays, itemCount int) error {
	_, err := uc.deliver.Execute(ctx, CreateNotificationInput{
		Type: notificationsDomain.TypeStockCloseToExpiry, Params: map[string]any{"count": itemCount, "window_days": windowDays}, Link: "/inventory",
	})
	if err != nil {
		return fmt.Errorf("notifyStockCloseToExpiry: %w", err)
	}
	return nil
}
