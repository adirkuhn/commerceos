package pms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestUpdateProduct_rejectsDisablingHasVariantsWithMultipleVariants(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{existing: []pmsDomain.Variant{{ID: "v1", ProductID: "p1"}, {ID: "v2", ProductID: "p1"}}}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, HasVariants: false})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrVariantsNotEnabled))
}

func TestUpdateProduct_allowsDisablingHasVariantsWithAtMostOneVariant(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{existing: []pmsDomain.Variant{{ID: "v1", ProductID: "p1"}}}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	product, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, HasVariants: false})

	require.NoError(t, err)
	require.False(t, product.HasVariants)
}

func TestUpdateProduct_allowsEnablingHasVariantsRegardlessOfVariantCount(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: false, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{existing: nil}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	product, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, HasVariants: true})

	require.NoError(t, err)
	require.True(t, product.HasVariants)
}

// TestUpdateProduct_flippingToVariantsRelinksTheExistingProductLevelListing
// is the fix for a real production bug: enabling variants on a product that
// was already pushed to WooCommerce (as a flat, single-SKU product) left
// its channel listing keyed VariantID=nil while the next push started
// looking it up by the variant's own ID, found nothing, and tried to
// re-CREATE the same SKU on Woo -- which Woo rejects as a duplicate, since
// it already has the original product. The fix relinks the existing
// product-level row (VariantID nil) to the one variant that already
// existed before the flip, instead of leaving it orphaned.
func TestUpdateProduct_flippingToVariantsRelinksTheExistingProductLevelListing(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: false, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{existing: []pmsDomain.Variant{{ID: "v1", ProductID: "p1"}}}
	listingRepo := &stubProductListingRepo{byProduct: []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: "p1", ConnectionID: "conn-1", ExternalID: "23609", VariantID: nil},
	}}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, listingRepo, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, HasVariants: true})

	require.NoError(t, err)
	require.Len(t, listingRepo.updated, 1)
	require.NotNil(t, listingRepo.updated[0].VariantID)
	require.Equal(t, "v1", *listingRepo.updated[0].VariantID)
	require.Equal(t, "listing-1", listingRepo.updated[0].ID)
}

// TestUpdateProduct_flippingToVariantsWithMultipleExistingVariantsDoesNotRelink
// guards the ambiguous case: relinking only makes sense for the flat-product
// shape (exactly one variant already existed) -- if the product somehow has
// more than one variant already at the moment of the flip, there's no single
// "the" variant to relink an existing product-level listing to, so it's left
// untouched rather than guessing.
func TestUpdateProduct_flippingToVariantsWithMultipleExistingVariantsDoesNotRelink(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: false, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{existing: []pmsDomain.Variant{{ID: "v1", ProductID: "p1"}, {ID: "v2", ProductID: "p1"}}}
	listingRepo := &stubProductListingRepo{byProduct: []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: "p1", ConnectionID: "conn-1", ExternalID: "23609", VariantID: nil},
	}}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, listingRepo, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, HasVariants: true})

	require.NoError(t, err)
	require.Empty(t, listingRepo.updated)
}

// TestUpdateProduct_alreadyInVariantsModeDoesNotRelinkOnEveryUnrelatedSave
// guards against relinking firing on every ordinary save of an
// already-variants-mode product -- it must only run on the false->true
// transition, once.
func TestUpdateProduct_alreadyInVariantsModeDoesNotRelinkOnEveryUnrelatedSave(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{existing: []pmsDomain.Variant{{ID: "v1", ProductID: "p1"}}}
	listingRepo := &stubProductListingRepo{byProduct: []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: "p1", ConnectionID: "conn-1", ExternalID: "23609", VariantID: nil},
	}}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, listingRepo, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, HasVariants: true})

	require.NoError(t, err)
	require.Empty(t, listingRepo.updated)
}

func TestUpdateProduct_updatesTags(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", Tags: []string{"old"}, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	product, err := uc.Execute(context.Background(), apppms.UpdateProductInput{
		ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, Tags: []string{"summer", "sale"},
	})

	require.NoError(t, err)
	require.Equal(t, []string{"summer", "sale"}, product.Tags)
}

// TestUpdateProduct_updatesSlugWhenProvided proves an explicit slug edit
// persists -- this is the fix for duplicated products being permanently
// stuck with their "-copy" slug: DuplicateProductUseCase gives the copy a
// "-copy" slug for uniqueness at creation time, and until now there was no
// way to ever change a product's slug afterward.
func TestUpdateProduct_updatesSlugWhenProvided(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", Slug: "biscoito-de-maizena-renata-360g-copy", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	newSlug := "biscoito-de-maizena-renata-360g"
	product, err := uc.Execute(context.Background(), apppms.UpdateProductInput{
		ID: "p1", Name: "Biscoito de Maizena - Renata - 360g", Status: pmsDomain.ProductStatusActive, Slug: &newSlug,
	})

	require.NoError(t, err)
	require.Equal(t, newSlug, product.Slug)
}

func TestUpdateProduct_leavesSlugUnchangedWhenNotProvided(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", Slug: "t-shirt", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	product, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive})

	require.NoError(t, err)
	require.Equal(t, "t-shirt", product.Slug)
}

func TestUpdateProduct_rejectsSlugAlreadyUsedByAnotherProduct(t *testing.T) {
	productRepo := &stubProductRepo{
		product:   &pmsDomain.Product{ID: "p1", Slug: "t-shirt", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		slugTaken: "other-product",
		slugOwner: &pmsDomain.Product{ID: "p2", Slug: "other-product"},
	}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	newSlug := "other-product"
	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, Slug: &newSlug})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrDuplicateSlug))
}

// TestUpdateProduct_allowsReSubmittingItsOwnCurrentSlug proves the
// uniqueness check doesn't false-positive against the product's own
// existing slug (FindBySlug would return the same product back).
func TestUpdateProduct_allowsReSubmittingItsOwnCurrentSlug(t *testing.T) {
	productRepo := &stubProductRepo{
		product:   &pmsDomain.Product{ID: "p1", Slug: "t-shirt", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		slugTaken: "t-shirt",
		slugOwner: &pmsDomain.Product{ID: "p1", Slug: "t-shirt"},
	}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	sameSlug := "t-shirt"
	product, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, Slug: &sameSlug})

	require.NoError(t, err)
	require.Equal(t, "t-shirt", product.Slug)
}

// TestUpdateProduct_regeneratesSlugFromNameWhenSlugCleared proves an
// explicitly empty slug regenerates from Name (same as create) rather than
// being rejected as malformed -- the empty string is never persisted.
func TestUpdateProduct_regeneratesSlugFromNameWhenSlugCleared(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", Name: "T-Shirt", Slug: "t-shirt", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	emptySlug := ""
	product, err := uc.Execute(context.Background(), apppms.UpdateProductInput{
		ID: "p1", Name: "Biscoito de Maizena - Renata - 360g", Status: pmsDomain.ProductStatusActive, Slug: &emptySlug,
	})

	require.NoError(t, err)
	require.Equal(t, "biscoito-de-maizena-renata-360g", product.Slug)
}

// TestUpdateProduct_clearingSlugWithUnchangedNameIsANoop proves regenerating
// from an unchanged Name, when it happens to match the current slug, skips
// the uniqueness check entirely rather than false-positiving against the
// product's own row.
func TestUpdateProduct_clearingSlugWithUnchangedNameIsANoop(t *testing.T) {
	productRepo := &stubProductRepo{
		product:   &pmsDomain.Product{ID: "p1", Name: "T-Shirt", Slug: "t-shirt", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		slugTaken: "should-not-be-queried",
	}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	emptySlug := ""
	product, err := uc.Execute(context.Background(), apppms.UpdateProductInput{
		ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, Slug: &emptySlug,
	})

	require.NoError(t, err)
	require.Equal(t, "t-shirt", product.Slug)
}

// TestUpdateProduct_regeneratedSlugAutoDisambiguatesOnCollision proves a
// regenerated (empty -> derived-from-Name) slug that collides with another
// product gets a numeric suffix instead of bouncing the save back with a
// conflict -- a cleared slug carries no user intent about its exact text,
// unlike one the user typed themselves (see the malformed/explicit-collision
// tests below, which still error).
func TestUpdateProduct_regeneratedSlugAutoDisambiguatesOnCollision(t *testing.T) {
	productRepo := &stubProductRepo{
		product:   &pmsDomain.Product{ID: "p1", Name: "T-Shirt", Slug: "t-shirt", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		slugTaken: "other-product",
		slugOwner: &pmsDomain.Product{ID: "p2", Slug: "other-product"},
	}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	emptySlug := ""
	product, err := uc.Execute(context.Background(), apppms.UpdateProductInput{
		ID: "p1", Name: "Other Product", Status: pmsDomain.ProductStatusActive, Slug: &emptySlug,
	})

	require.NoError(t, err)
	require.Equal(t, "other-product-2", product.Slug)
}

// TestUpdateProduct_explicitSlugStillRejectsCollision proves the
// auto-disambiguation above is specific to a *regenerated* slug -- a slug
// the user typed themselves still errors on collision rather than silently
// saving under different text than what they entered.
func TestUpdateProduct_explicitSlugStillRejectsCollision(t *testing.T) {
	productRepo := &stubProductRepo{
		product:   &pmsDomain.Product{ID: "p1", Name: "T-Shirt", Slug: "t-shirt", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		slugTaken: "other-product",
		slugOwner: &pmsDomain.Product{ID: "p2", Slug: "other-product"},
	}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	typedSlug := "other-product"
	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{
		ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, Slug: &typedSlug,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrDuplicateSlug))
}

func TestUpdateProduct_rejectsMalformedSlug(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", Slug: "t-shirt", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, &stubOutboxRepo{}, &stubProductHistoryRepo{})

	badSlug := "Not A Valid Slug!"
	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, Slug: &badSlug})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrInvalidSlug))
}

func TestUpdateProduct_raisesProductChangedEvent(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: false, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{existing: nil}
	outboxRepo := &stubOutboxRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubProductListingRepo{}, outboxRepo, &stubProductHistoryRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive})

	require.NoError(t, err)
	require.Equal(t, []string{"pms.product.changed"}, outboxRepo.calls)
}

func TestUpdateProduct_recordsHistoryForNameAndStatusChanges(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{
		ID: "p1", Name: "Old Name", Status: pmsDomain.ProductStatusDraft, HasVariants: false, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}}
	historyRepo := &stubProductHistoryRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, &stubVariantRepo{}, &stubProductListingRepo{}, &stubOutboxRepo{}, historyRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "New Name", Status: pmsDomain.ProductStatusActive})

	require.NoError(t, err)
	require.Len(t, historyRepo.created, 2)
	byField := map[pmsDomain.ProductHistoryField]pmsDomain.ProductHistory{}
	for _, h := range historyRepo.created {
		byField[h.Field] = h
	}
	require.Equal(t, "Old Name", byField[pmsDomain.ProductHistoryFieldName].OldValue)
	require.Equal(t, "New Name", byField[pmsDomain.ProductHistoryFieldName].NewValue)
	require.Equal(t, pmsDomain.ProductHistoryEntityProduct, byField[pmsDomain.ProductHistoryFieldName].EntityType)
	require.Equal(t, "p1", byField[pmsDomain.ProductHistoryFieldName].EntityID)
	require.Equal(t, "draft", byField[pmsDomain.ProductHistoryFieldStatus].OldValue)
	require.Equal(t, "active", byField[pmsDomain.ProductHistoryFieldStatus].NewValue)
}

func TestUpdateProduct_doesNotRecordHistoryWhenNameAndStatusUnchanged(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{
		ID: "p1", Name: "Same Name", Status: pmsDomain.ProductStatusActive, HasVariants: false, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}}
	historyRepo := &stubProductHistoryRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, &stubVariantRepo{}, &stubProductListingRepo{}, &stubOutboxRepo{}, historyRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "Same Name", Status: pmsDomain.ProductStatusActive, Description: "only this changed"})

	require.NoError(t, err)
	require.Empty(t, historyRepo.created)
}

func TestUpdateProduct_propagatesHistoryRepoError(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", Name: "Old Name", Status: pmsDomain.ProductStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	historyRepo := &stubProductHistoryRepo{createErr: errors.New("db down")}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, &stubVariantRepo{}, &stubProductListingRepo{}, &stubOutboxRepo{}, historyRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "New Name", Status: pmsDomain.ProductStatusDraft})

	require.Error(t, err)
}
