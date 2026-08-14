package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestDeleteAllImportedProducts_deletesAllProductsAndEmptyCategory(t *testing.T) {
	h := newHarness(nil)
	catID := "cat-1"
	h.productRepo.byID = map[string]pmsDomain.Product{
		"p1": {ID: "p1", CategoryID: &catID},
		"p2": {ID: "p2", CategoryID: &catID},
	}
	h.productRepo.countByCategory = map[string]int64{"cat-1": 0}
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{
		{ID: "l1", ProductID: "p1", ConnectionID: "conn-1"},
		{ID: "l2", ProductID: "p2", ConnectionID: "conn-1"},
	}

	uc := appchannels.NewDeleteAllImportedProductsUseCase(newMockGormDB(t, 4), h.listingRepo, h.productRepo, h.categoryRepo)

	result, err := uc.Execute(context.Background(), appchannels.DeleteAllImportedProductsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 2, result.DeletedProducts)
	require.Equal(t, 1, result.DeletedCategories)
	require.Empty(t, result.Errors)
	require.ElementsMatch(t, []string{"p1", "p2"}, h.productRepo.deleted)
	require.Equal(t, []string{"cat-1"}, h.categoryRepo.deleted)
}

func TestDeleteAllImportedProducts_leavesCategoryStillUsedByOtherProducts(t *testing.T) {
	h := newHarness(nil)
	catID := "cat-1"
	h.productRepo.byID = map[string]pmsDomain.Product{"p1": {ID: "p1", CategoryID: &catID}}
	h.productRepo.countByCategory = map[string]int64{"cat-1": 3}
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{
		{ID: "l1", ProductID: "p1", ConnectionID: "conn-1"},
	}

	uc := appchannels.NewDeleteAllImportedProductsUseCase(newMockGormDB(t, 3), h.listingRepo, h.productRepo, h.categoryRepo)

	result, err := uc.Execute(context.Background(), appchannels.DeleteAllImportedProductsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 1, result.DeletedProducts)
	require.Equal(t, 0, result.DeletedCategories)
	require.Empty(t, h.categoryRepo.deleted)
}

func TestDeleteAllImportedProducts_skipsCategoryStepWhenProductHasNone(t *testing.T) {
	h := newHarness(nil)
	h.productRepo.byID = map[string]pmsDomain.Product{"p1": {ID: "p1", CategoryID: nil}}
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{
		{ID: "l1", ProductID: "p1", ConnectionID: "conn-1"},
	}

	uc := appchannels.NewDeleteAllImportedProductsUseCase(newMockGormDB(t, 2), h.listingRepo, h.productRepo, h.categoryRepo)

	result, err := uc.Execute(context.Background(), appchannels.DeleteAllImportedProductsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 1, result.DeletedProducts)
	require.Equal(t, 0, result.DeletedCategories)
}

func TestDeleteAllImportedProducts_recordsErrorPerProductAndContinues(t *testing.T) {
	h := newHarness(nil)
	h.productRepo.byID = map[string]pmsDomain.Product{"p2": {ID: "p2"}} // p1 missing -> FindByID fails
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{
		{ID: "l1", ProductID: "p1", ConnectionID: "conn-1"},
		{ID: "l2", ProductID: "p2", ConnectionID: "conn-1"},
	}

	uc := appchannels.NewDeleteAllImportedProductsUseCase(newMockGormDB(t, 3), h.listingRepo, h.productRepo, h.categoryRepo)

	result, err := uc.Execute(context.Background(), appchannels.DeleteAllImportedProductsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 1, result.DeletedProducts)
	require.Len(t, result.Errors, 1)
	require.Equal(t, []string{"p2"}, h.productRepo.deleted)
}

// TestDeleteAllImportedProducts_reportsCleanMessageForReceivingHistory proves
// a product blocked by booking_scans (ON DELETE RESTRICT, see
// pmsDomain.ErrProductHasReceivingHistory) surfaces its clean typed message
// in result.Errors, not a raw SQL/constraint string, while every other
// product still deletes fine.
func TestDeleteAllImportedProducts_reportsCleanMessageForReceivingHistory(t *testing.T) {
	h := newHarness(nil)
	h.productRepo.byID = map[string]pmsDomain.Product{"p1": {ID: "p1"}, "p2": {ID: "p2"}}
	h.productRepo.deleteErr = pmsDomain.ErrProductHasReceivingHistory
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{
		{ID: "l1", ProductID: "p1", ConnectionID: "conn-1"},
	}

	uc := appchannels.NewDeleteAllImportedProductsUseCase(newMockGormDB(t, 2), h.listingRepo, h.productRepo, h.categoryRepo)

	result, err := uc.Execute(context.Background(), appchannels.DeleteAllImportedProductsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 0, result.DeletedProducts)
	require.Len(t, result.Errors, 1)
	require.Contains(t, result.Errors[0], pmsDomain.ErrProductHasReceivingHistory.Error())
	require.NotContains(t, result.Errors[0], "SQLSTATE", "must not leak the raw Postgres error")
}

func TestDeleteAllImportedProducts_recordsErrorWhenCategoryDeleteFails(t *testing.T) {
	h := newHarness(nil)
	catID := "cat-1"
	h.productRepo.byID = map[string]pmsDomain.Product{"p1": {ID: "p1", CategoryID: &catID}}
	h.productRepo.countByCategory = map[string]int64{"cat-1": 0}
	h.categoryRepo.deleteErr = errors.New("db down")
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{
		{ID: "l1", ProductID: "p1", ConnectionID: "conn-1"},
	}

	uc := appchannels.NewDeleteAllImportedProductsUseCase(newMockGormDB(t, 3), h.listingRepo, h.productRepo, h.categoryRepo)

	result, err := uc.Execute(context.Background(), appchannels.DeleteAllImportedProductsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 1, result.DeletedProducts)
	require.Equal(t, 0, result.DeletedCategories)
	require.Len(t, result.Errors, 1)
}

func TestDeleteAllImportedProducts_noopWhenNoListings(t *testing.T) {
	h := newHarness(nil)

	uc := appchannels.NewDeleteAllImportedProductsUseCase(newMockGormDB(t, 1), h.listingRepo, h.productRepo, h.categoryRepo)

	result, err := uc.Execute(context.Background(), appchannels.DeleteAllImportedProductsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 0, result.DeletedProducts)
	require.Equal(t, 0, result.DeletedCategories)
}

func TestDeleteAllImportedProducts_propagatesListingListError(t *testing.T) {
	h := newHarness(nil)
	h.listingRepo.findErr = errors.New("db down")

	uc := appchannels.NewDeleteAllImportedProductsUseCase(newMockGormDB(t, 1), h.listingRepo, h.productRepo, h.categoryRepo)

	_, err := uc.Execute(context.Background(), appchannels.DeleteAllImportedProductsInput{ConnectionID: "conn-1"})

	require.Error(t, err)
}
