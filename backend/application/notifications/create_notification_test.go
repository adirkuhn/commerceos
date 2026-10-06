package notifications_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestCreateNotification_setsTenantAndFields(t *testing.T) {
	repo := &stubNotificationRepo{}
	uc := appnotifications.NewCreateNotificationUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	n, err := uc.Execute(ctx, appnotifications.CreateNotificationInput{
		Type: notificationsDomain.TypeImportStarted, Title: "Import started", Message: "Importing…",
	})

	require.NoError(t, err)
	require.Equal(t, "tenant-1", n.TenantID)
	require.Equal(t, notificationsDomain.TypeImportStarted, n.Type)
	require.Equal(t, "Import started", n.Title)
	require.Len(t, repo.created, 1)
}

func TestCreateNotification_propagatesRepoError(t *testing.T) {
	repo := &stubNotificationRepo{createErr: errors.New("db down")}
	uc := appnotifications.NewCreateNotificationUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appnotifications.CreateNotificationInput{Title: "X"})

	require.Error(t, err)
}
