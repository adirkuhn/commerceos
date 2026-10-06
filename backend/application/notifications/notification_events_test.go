package notifications_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func only(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

func TestCreateNotification_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appnotifications.NewCreateNotificationUseCase(newMockGormDB(t), &stubNotificationRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appnotifications.CreateNotificationInput{Type: notificationsDomain.Type("low_stock"), Title: "Low stock", Link: "/inventory"})

	require.NoError(t, err)
	e := only(t, outbox)
	assert.Equal(t, "notifications.notification.created", e.Subject)
	assert.Equal(t, "notif-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"type": "low_stock", "title": "Low stock", "link": "/inventory"}, e.Data())
}

func TestMarkNotificationRead_RecordsRead(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appnotifications.NewMarkNotificationReadUseCase(newMockGormDB(t), &stubNotificationRepo{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "notif-9"))

	e := only(t, outbox)
	assert.Equal(t, "notifications.notification.read", e.Subject)
	assert.Equal(t, "notif-9", e.Payload["entity_id"])
}

func TestMarkAllNotificationsRead_RecordsAllRead(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appnotifications.NewMarkAllNotificationsReadUseCase(newMockGormDB(t), &stubNotificationRepo{}, outbox)

	require.NoError(t, uc.Execute(tenant.WithTenantID(context.Background(), "t-1")))

	e := only(t, outbox)
	assert.Equal(t, "notifications.notification.all_read", e.Subject)
	assert.Equal(t, "t-1", e.Payload["entity_id"])
}

func TestNotificationUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	_, err := appnotifications.NewCreateNotificationUseCase(newMockGormDB(t), &stubNotificationRepo{}, failing).Execute(context.Background(), appnotifications.CreateNotificationInput{})
	assert.ErrorIs(t, err, boom)
	assert.ErrorIs(t, appnotifications.NewMarkNotificationReadUseCase(newMockGormDB(t), &stubNotificationRepo{}, failing).Execute(context.Background(), "n"), boom)
	assert.ErrorIs(t, appnotifications.NewMarkAllNotificationsReadUseCase(newMockGormDB(t), &stubNotificationRepo{}, failing).Execute(context.Background()), boom)
}
