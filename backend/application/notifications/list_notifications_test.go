package notifications_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
)

func TestListNotifications_returnsRepoResult(t *testing.T) {
	repo := &stubNotificationRepo{listResult: []notificationsDomain.Notification{{ID: "n1", Title: "Import started"}}}
	uc := appnotifications.NewListNotificationsUseCase(newMockGormDB(t), repo)

	out, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "Import started", out[0].Title)
}

func TestListNotifications_propagatesRepoError(t *testing.T) {
	repo := &stubNotificationRepo{listErr: errors.New("db down")}
	uc := appnotifications.NewListNotificationsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}
