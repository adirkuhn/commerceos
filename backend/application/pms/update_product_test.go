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

func TestUpdateProduct_raisesProductChangedEvent(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: false, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	variantRepo := &stubVariantRepo{existing: nil}
	outboxRepo := &stubOutboxRepo{}
	uc := apppms.NewUpdateProductUseCase(newMockGormDB(t), productRepo, variantRepo, outboxRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateProductInput{ID: "p1", Name: "T-Shirt", Status: pmsDomain.ProductStatusActive})

	require.NoError(t, err)
	require.Equal(t, []string{"pms.product.changed"}, outboxRepo.calls)
}
