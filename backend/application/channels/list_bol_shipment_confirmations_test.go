package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func TestListBolShipmentConfirmations_returnsTheOrdersConfirmations(t *testing.T) {
	want := []channelsDomain.BolShipmentConfirmation{{ID: "conf-2"}, {ID: "conf-1"}}
	uc := appchannels.NewListBolShipmentConfirmationsUseCase(newMockGormDB(t, 1),
		&stubOrdersSalesOrderRepo{findByIDResult: shippedBolOrder()}, &fakeBolConfirmationRepo{existing: want})

	got, err := uc.Execute(context.Background(), "so-1")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestListBolShipmentConfirmations_noneYet_returnsEmptyNotNil(t *testing.T) {
	uc := appchannels.NewListBolShipmentConfirmationsUseCase(newMockGormDB(t, 1),
		&stubOrdersSalesOrderRepo{findByIDResult: shippedBolOrder()}, &fakeBolConfirmationRepo{})

	got, err := uc.Execute(context.Background(), "so-1")

	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestListBolShipmentConfirmations_unknownOrder_isNotFound(t *testing.T) {
	uc := appchannels.NewListBolShipmentConfirmationsUseCase(newMockGormDB(t, 1),
		&stubOrdersSalesOrderRepo{}, &fakeBolConfirmationRepo{})

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}

func TestListBolShipmentConfirmations_repoError_isReturned(t *testing.T) {
	uc := appchannels.NewListBolShipmentConfirmationsUseCase(newMockGormDB(t, 1),
		&stubOrdersSalesOrderRepo{findByIDResult: shippedBolOrder()}, &fakeBolConfirmationRepo{listErr: errors.New("db down")})

	_, err := uc.Execute(context.Background(), "so-1")

	assert.Error(t, err)
}
