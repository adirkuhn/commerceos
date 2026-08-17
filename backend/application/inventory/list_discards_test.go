package inventory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestListDiscards_returnsTheReportFromTheRepo(t *testing.T) {
	discardRepo := &stubDiscardRecordRepo{
		list:      []invDomain.DiscardRecordReport{{DiscardRecord: invDomain.DiscardRecord{ID: "d1"}, ProductName: "Widget"}},
		listTotal: 1,
	}
	uc := appinv.NewListDiscardsUseCase(newMockGormDB(t), discardRepo)

	result, err := uc.Execute(context.Background(), appinv.ListDiscardsInput{})

	require.NoError(t, err)
	require.Len(t, result.Discards, 1)
	require.Equal(t, "Widget", result.Discards[0].ProductName)
	require.Equal(t, int64(1), result.Total)
}

func TestListDiscards_defaultsAndCapsLimit(t *testing.T) {
	discardRepo := &stubDiscardRecordRepo{}
	uc := appinv.NewListDiscardsUseCase(newMockGormDBN(t, 2), discardRepo)

	_, err := uc.Execute(context.Background(), appinv.ListDiscardsInput{Limit: 0})
	require.NoError(t, err)
	require.Equal(t, 50, discardRepo.gotFilter.Limit, "zero limit should default")

	_, err = uc.Execute(context.Background(), appinv.ListDiscardsInput{Limit: 9999})
	require.NoError(t, err)
	require.Equal(t, 200, discardRepo.gotFilter.Limit, "limit should be capped")
}

func TestListDiscards_passesWarehouseSKUAndDateFiltersThrough(t *testing.T) {
	discardRepo := &stubDiscardRecordRepo{}
	uc := appinv.NewListDiscardsUseCase(newMockGormDB(t), discardRepo)
	wh, sku := "wh-1", "sku-1"

	_, err := uc.Execute(context.Background(), appinv.ListDiscardsInput{WarehouseID: &wh, SKUID: &sku})

	require.NoError(t, err)
	require.Equal(t, &wh, discardRepo.gotFilter.WarehouseID)
	require.Equal(t, &sku, discardRepo.gotFilter.SKUID)
}

func TestListDiscards_propagatesRepoError(t *testing.T) {
	discardRepo := &stubDiscardRecordRepo{listErr: invDomain.ErrBatchNotFound}
	uc := appinv.NewListDiscardsUseCase(newMockGormDB(t), discardRepo)

	_, err := uc.Execute(context.Background(), appinv.ListDiscardsInput{})

	require.Error(t, err)
}
