package pms_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type fakeUnlistListings struct {
	byProduct []channelsDomain.ProductChannelListing
	deleted   []string
}

func (f *fakeUnlistListings) ListByProduct(_ context.Context, _ string) ([]channelsDomain.ProductChannelListing, error) {
	return f.byProduct, nil
}
func (f *fakeUnlistListings) Delete(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func carneSecaListings() *fakeUnlistListings {
	v222, v380 := "variant-222", "variant-380"
	return &fakeUnlistListings{byProduct: []channelsDomain.ProductChannelListing{
		{ID: "l-222", ConnectionID: "woo", Provider: "woocommerce", ExternalID: "901", VariantID: &v222},
		{ID: "l-380", ConnectionID: "woo", Provider: "woocommerce", ExternalID: "900", VariantID: &v380},
	}}
}

func unlistedEvent(t *testing.T, outbox *stubOutboxRepo) events.VariantUnlisted {
	t.Helper()
	for i, subject := range outbox.calls {
		if subject == "pms.variant.unlisted" {
			var e events.VariantUnlisted
			require.NoError(t, json.Unmarshal(outbox.payloads[i], &e))
			return e
		}
	}
	t.Fatalf("no pms.variant.unlisted enqueued, got %v", outbox.calls)
	return events.VariantUnlisted{}
}

func TestDeleteVariant_takesItOffEveryChannelItWasListedOn(t *testing.T) {
	listings := carneSecaListings()
	outbox := &stubOutboxRepo{}
	variants := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-222", ProductID: "product-1"}}
	uc := apppms.NewDeleteVariantUseCase(newMockGormDB(t), variants, listings, outbox)

	require.NoError(t, uc.Execute(context.Background(), "variant-222"))

	e := unlistedEvent(t, outbox)
	assert.Equal(t, "product-1", e.ProductID)
	assert.Equal(t, "variant-222", e.VariantID)
	assert.Equal(t, []events.VariantUnlistedListing{{ConnectionID: "woo", Provider: "woocommerce", ExternalID: "901"}}, e.Listings)
	assert.Contains(t, outbox.calls, "pms.product.changed", "siblings/parent resync")
}

func TestDeleteVariant_neverListedRaisesNoUnlistedEvent(t *testing.T) {
	outbox := &stubOutboxRepo{}
	variants := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-new", ProductID: "product-1"}}
	uc := apppms.NewDeleteVariantUseCase(newMockGormDB(t), variants, carneSecaListings(), outbox)

	require.NoError(t, uc.Execute(context.Background(), "variant-new"))

	assert.NotContains(t, outbox.calls, "pms.variant.unlisted")
}

func TestDeleteSKU_deletingAVariantsLastSKUUnlistsTheVariant(t *testing.T) {
	listings := carneSecaListings()
	outbox := &stubOutboxRepo{}
	sku := pmsDomain.SKU{ID: "sku-222", VariantID: "variant-222"}
	skus := &stubSKURepo{findByIDResult: &sku, listByVariantResult: []pmsDomain.SKU{sku}}
	variants := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-222", ProductID: "product-1"}}
	uc := apppms.NewDeleteSKUUseCase(newMockGormDB(t), skus, variants, listings, outbox)

	require.NoError(t, uc.Execute(context.Background(), "sku-222"))

	assert.Equal(t, []string{"l-222"}, listings.deleted, "the variant stays, so its now-empty listing row must go too")
	e := unlistedEvent(t, outbox)
	assert.Equal(t, "variant-222", e.VariantID)
	assert.Equal(t, "901", e.Listings[0].ExternalID)
}

func TestDeleteSKU_aVariantWithAnotherSKULeftStaysListed(t *testing.T) {
	listings := carneSecaListings()
	outbox := &stubOutboxRepo{}
	sku := pmsDomain.SKU{ID: "sku-222", VariantID: "variant-222"}
	other := pmsDomain.SKU{ID: "sku-222-b", VariantID: "variant-222"}
	skus := &stubSKURepo{findByIDResult: &sku, listByVariantResult: []pmsDomain.SKU{sku, other}}
	variants := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-222", ProductID: "product-1"}}
	uc := apppms.NewDeleteSKUUseCase(newMockGormDB(t), skus, variants, listings, outbox)

	require.NoError(t, uc.Execute(context.Background(), "sku-222"))

	assert.Empty(t, listings.deleted)
	assert.NotContains(t, outbox.calls, "pms.variant.unlisted")
	assert.Contains(t, outbox.calls, "pms.product.changed")
}
