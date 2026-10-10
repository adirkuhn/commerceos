package notifications_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
)

type stubDeliverer struct {
	got []appnotifications.CreateNotificationInput
	err error
}

func (s *stubDeliverer) Execute(_ context.Context, in appnotifications.CreateNotificationInput) (*notificationsDomain.Notification, error) {
	s.got = append(s.got, in)
	return nil, s.err
}

func TestNotifyStockCloseToExpiry_raisesABellNotification(t *testing.T) {
	d := &stubDeliverer{}
	uc := appnotifications.NewNotifyStockCloseToExpiryUseCase(d)

	require.NoError(t, uc.Execute(tctx(), 30, 4))
	require.NoError(t, uc.Execute(tctx(), 7, 1))

	assert.Equal(t, []appnotifications.CreateNotificationInput{
		{Type: notificationsDomain.TypeStockCloseToExpiry, Title: "Stock close to expiry", Message: "4 products expire within 30 days", Link: "/inventory"},
		{Type: notificationsDomain.TypeStockCloseToExpiry, Title: "Stock close to expiry", Message: "1 product expires within 7 days", Link: "/inventory"},
	}, d.got)
}

func TestNotifyStockCloseToExpiry_propagatesError(t *testing.T) {
	boom := errors.New("boom")
	uc := appnotifications.NewNotifyStockCloseToExpiryUseCase(&stubDeliverer{err: boom})

	assert.ErrorIs(t, uc.Execute(tctx(), 30, 4), boom)
}
