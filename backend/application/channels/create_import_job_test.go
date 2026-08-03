package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestCreateImportJob_createsPendingJobAndStartedNotification(t *testing.T) {
	jobRepo := &stubImportJobRepo{}
	notif := &stubNotifier{}
	uc := appchannels.NewCreateImportJobUseCase(newMockGormDB(t, 1), jobRepo, notif)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	job, err := uc.Execute(ctx, appchannels.CreateImportJobInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, channelsDomain.ImportJobPending, job.Status)
	require.Equal(t, "conn-1", job.ConnectionID)
	require.Equal(t, "tenant-1", job.TenantID)
	require.Len(t, jobRepo.created, 1)
	require.Len(t, notif.calls, 1)
	require.Contains(t, notif.calls[0].Title, "started")
}

func TestCreateImportJob_rejectsWhenAlreadyActive(t *testing.T) {
	jobRepo := &stubImportJobRepo{hasActive: true}
	notif := &stubNotifier{}
	uc := appchannels.NewCreateImportJobUseCase(newMockGormDB(t, 1), jobRepo, notif)

	_, err := uc.Execute(context.Background(), appchannels.CreateImportJobInput{ConnectionID: "conn-1"})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrImportJobAlreadyActive))
	require.Empty(t, jobRepo.created)
	require.Empty(t, notif.calls)
}

func TestCreateImportJob_propagatesCreateError(t *testing.T) {
	jobRepo := &stubImportJobRepo{createErr: errors.New("db down")}
	notif := &stubNotifier{}
	uc := appchannels.NewCreateImportJobUseCase(newMockGormDB(t, 1), jobRepo, notif)

	_, err := uc.Execute(context.Background(), appchannels.CreateImportJobInput{ConnectionID: "conn-1"})

	require.Error(t, err)
	require.Empty(t, notif.calls)
}

func TestCreateImportJob_succeedsEvenIfNotificationFails(t *testing.T) {
	jobRepo := &stubImportJobRepo{}
	notif := &stubNotifier{err: errors.New("notif db down")}
	uc := appchannels.NewCreateImportJobUseCase(newMockGormDB(t, 1), jobRepo, notif)

	job, err := uc.Execute(context.Background(), appchannels.CreateImportJobInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.NotNil(t, job)
	require.Len(t, jobRepo.created, 1)
}
