package notifications_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
)

func TestMarkNotificationRead_callsRepoWithID(t *testing.T) {
	repo := &stubNotificationRepo{}
	uc := appnotifications.NewMarkNotificationReadUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "notif-1")

	require.NoError(t, err)
	require.Equal(t, "notif-1", repo.markReadID)
}

func TestMarkNotificationRead_propagatesRepoError(t *testing.T) {
	repo := &stubNotificationRepo{markReadErr: errors.New("db down")}
	uc := appnotifications.NewMarkNotificationReadUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "notif-1")

	require.Error(t, err)
}

func TestMarkAllNotificationsRead_callsRepo(t *testing.T) {
	repo := &stubNotificationRepo{}
	uc := appnotifications.NewMarkAllNotificationsReadUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.True(t, repo.markAllReadCalled)
}

func TestMarkAllNotificationsRead_propagatesRepoError(t *testing.T) {
	repo := &stubNotificationRepo{markAllReadErr: errors.New("db down")}
	uc := appnotifications.NewMarkAllNotificationsReadUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background())

	require.Error(t, err)
}
