package pms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestUpdateProduct_rejectsDisablingHasVariantsWithMultipleVariants(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{existing: []pmsDomain.Variant{{ID: "v1", ProductID: "p1"}, {ID: "v2", ProductID: "p1"}}}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, HasVariants: false})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrVariantsNotEnabled))
}

func TestUpdateProduct_allowsDisablingHasVariantsWithAtMostOneVariant(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{existing: []pmsDomain.Variant{{ID: "v1", ProductID: "p1"}}}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

	product, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, HasVariants: false})

	require.NoError(t, err)
	require.False(t, product.HasVariants)
}

func TestUpdateProduct_allowsEnablingHasVariantsRegardlessOfVariantCount(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: false, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{existing: nil}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

	product, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, HasVariants: true})

	require.NoError(t, err)
	require.True(t, product.HasVariants)
}

func TestUpdateProduct_updatesTags(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", Tags: []string{"old"}, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

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
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

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
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

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
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

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
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

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
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

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
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

	emptySlug := ""
	product, err := uc.Execute(context.Background(), apppms.UpdateProductInput{
		ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, Slug: &emptySlug,
	})

	require.NoError(t, err)
	require.Equal(t, "t-shirt", product.Slug)
}

// TestUpdateProduct_regeneratedSlugStillRejectsCollision proves a
// regenerated (not just an explicitly-typed) slug still goes through the
// same duplicate check as any other slug change.
func TestUpdateProduct_regeneratedSlugStillRejectsCollision(t *testing.T) {
	productRepo := &stubProductRepo{
		product:   &pmsDomain.Product{ID: "p1", Name: "T-Shirt", Slug: "t-shirt", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		slugTaken: "other-product",
		slugOwner: &pmsDomain.Product{ID: "p2", Slug: "other-product"},
	}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

	emptySlug := ""
	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{
		ID: "p1", Name: "Other Product", Status: pmsDomain.ProductStatusActive, Slug: &emptySlug,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrDuplicateSlug))
}

func TestUpdateProduct_rejectsMalformedSlug(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", Slug: "t-shirt", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

	badSlug := "Not A Valid Slug!"
	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive, Slug: &badSlug})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrInvalidSlug))
}

func TestUpdateProduct_raisesProductChangedEvent(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: false, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{existing: nil}
	outboxRepo := &stubOutboxRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, outboxRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive})

	require.NoError(t, err)
	require.Equal(t, []string{"pms.product.changed"}, outboxRepo.calls)
}
