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

type stubProductEnroller struct {
	called []string
	err    error
}

func (s *stubProductEnroller) Execute(_ context.Context, productID string) error {
	s.called = append(s.called, productID)
	return s.err
}

func newResyncTestHarness(t *testing.T, n int) (*appchannels.ResyncProductToChannelsUseCase, *stubChannelsVariantRepo, *stubChannelsSKURepo, *stubProductFieldsSyncer, *stubProductStockForceSyncer, *stubProductEnroller) {
	t.Helper()
	variantRepo := &stubChannelsVariantRepo{}
	skuRepo := &stubChannelsSKURepo{}
	syncFields := &stubProductFieldsSyncer{}
	syncStock := &stubProductStockForceSyncer{}
	enroller := &stubProductEnroller{}
	uc := appchannels.NewResyncProductToChannelsUseCase(newMockGormDB(t, n), variantRepo, skuRepo, syncFields, syncStock, enroller)
	return uc, variantRepo, skuRepo, syncFields, syncStock, enroller
}

func TestResyncProductToChannels_pushesFieldsThenForcesStockForEverySKU(t *testing.T) {
	uc, variantRepo, skuRepo, syncFields, syncStock, _ := newResyncTestHarness(t, 2)
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
	uc, _, _, syncFields, syncStock, _ := newResyncTestHarness(t, 2)
	syncFields.err = errors.New("woo unreachable")

	err := uc.Execute(context.Background(), "product-1")

	require.Error(t, err)
	require.Empty(t, syncStock.called, "stock must not be pushed when the fields push itself failed")
}

func TestResyncProductToChannels_oneSKUStockFailureDoesNotBlockTheOthers(t *testing.T) {
	uc, variantRepo, skuRepo, _, syncStock, _ := newResyncTestHarness(t, 2)
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

// TestResyncProductToChannels_alsoEnrollsChannelsWithNoListingYet is a
// regression test for the exact bug making "Resync to channels" a silent
// no-op for the prod Hipoglos product: syncFields (SyncProductToChannelsUseCase)
// only ever refreshes channels the product ALREADY has a
// ProductChannelListing row for -- it hard no-ops when there are zero
// listings (see its own doc comment: "A product with no listings is a
// cheap no-op"). That's correct for its real job (reacting to
// pms.product.changed on every edit, where most products aren't linked
// anywhere), but wrong for the user-facing "Resync to channels" button,
// whose whole point is "push this everywhere it should be" -- including a
// channel the product is enrolled in (e.g. via a manual
// SalesChannelProductOverride) but that never got a first listing created,
// which is exactly what happened to every Bol product before the
// CreateOffer/UpdateOffer wire-format fix: they were enrolled, but the
// first push always silently failed to produce a listing. Resync must also
// run the enrollment-aware push (EnrollProductInChannelsUseCase) so a
// zero-listing-but-enrolled product gets its first listing created, not
// just its existing ones refreshed.
func TestResyncProductToChannels_alsoEnrollsChannelsWithNoListingYet(t *testing.T) {
	uc, _, _, syncFields, _, enroller := newResyncTestHarness(t, 2)

	err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err)
	require.Equal(t, []string{"product-1"}, enroller.called, "resync must also run enrollment-aware push so a never-yet-linked channel gets its first listing created")
	require.Equal(t, []string{"product-1"}, syncFields.called)
}

func TestResyncProductToChannels_enrollFailureDoesNotBlockFieldsSync(t *testing.T) {
	uc, _, _, syncFields, _, enroller := newResyncTestHarness(t, 2)
	enroller.err = errors.New("bol unreachable")

	err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err, "one channel's enroll failure must not fail the whole resync, same tolerance as every other step here")
	require.Equal(t, []string{"product-1"}, syncFields.called)
}
