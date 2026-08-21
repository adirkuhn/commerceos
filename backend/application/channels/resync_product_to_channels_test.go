package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubProductFieldsSyncer struct {
	called []string
	err    error
}

func (s *stubProductFieldsSyncer) Execute(_ context.Context, productID string) error {
	s.called = append(s.called, productID)
	return s.err
}

type stubProductStockForceSyncer struct {
	called []string
	errs   map[string]error
}

func (s *stubProductStockForceSyncer) ForceExecute(_ context.Context, skuID string) error {
	s.called = append(s.called, skuID)
	return s.errs[skuID]
}

func newResyncTestHarness(t *testing.T, n int) (*appchannels.ResyncProductToChannelsUseCase, *stubChannelsVariantRepo, *stubChannelsSKURepo, *stubProductFieldsSyncer, *stubProductStockForceSyncer) {
	t.Helper()
	variantRepo := &stubChannelsVariantRepo{}
	skuRepo := &stubChannelsSKURepo{}
	syncFields := &stubProductFieldsSyncer{}
	syncStock := &stubProductStockForceSyncer{}
	uc := appchannels.NewResyncProductToChannelsUseCase(newMockGormDB(t, n), variantRepo, skuRepo, syncFields, syncStock)
	return uc, variantRepo, skuRepo, syncFields, syncStock
}

func TestResyncProductToChannels_pushesFieldsThenForcesStockForEverySKU(t *testing.T) {
	uc, variantRepo, skuRepo, syncFields, syncStock := newResyncTestHarness(t, 2)
	variantRepo.byProduct = map[string][]pmsDomain.Variant{
		"product-1": {{ID: "variant-1", ProductID: "product-1"}, {ID: "variant-2", ProductID: "product-1"}},
	}
	skuRepo.byVariant = map[string][]pmsDomain.SKU{
		"variant-1": {{ID: "sku-1", VariantID: "variant-1"}},
		"variant-2": {{ID: "sku-2", VariantID: "variant-2"}},
	}

	err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err)
	require.Equal(t, []string{"product-1"}, syncFields.called)
	require.ElementsMatch(t, []string{"sku-1", "sku-2"}, syncStock.called)
}

func TestResyncProductToChannels_stopsBeforeStockWhenFieldsSyncFails(t *testing.T) {
	uc, _, _, syncFields, syncStock := newResyncTestHarness(t, 2)
	syncFields.err = errors.New("woo unreachable")

	err := uc.Execute(context.Background(), "product-1")

	require.Error(t, err)
	require.Empty(t, syncStock.called, "stock must not be pushed when the fields push itself failed")
}

func TestResyncProductToChannels_oneSKUStockFailureDoesNotBlockTheOthers(t *testing.T) {
	uc, variantRepo, skuRepo, _, syncStock := newResyncTestHarness(t, 2)
	variantRepo.byProduct = map[string][]pmsDomain.Variant{
		"product-1": {{ID: "variant-1", ProductID: "product-1"}, {ID: "variant-2", ProductID: "product-1"}},
	}
	skuRepo.byVariant = map[string][]pmsDomain.SKU{
		"variant-1": {{ID: "sku-1", VariantID: "variant-1"}},
		"variant-2": {{ID: "sku-2", VariantID: "variant-2"}},
	}
	syncStock.errs = map[string]error{"sku-1": errors.New("woo unreachable")}

	err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err, "a single SKU's stock push failure must not fail the whole resync")
	require.ElementsMatch(t, []string{"sku-1", "sku-2"}, syncStock.called)
}
