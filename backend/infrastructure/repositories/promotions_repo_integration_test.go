//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// seedPromotionSKU creates a product/variant/SKU under ctx's tenant and
// returns the SKU's ID -- a promotion_skus row's FK requires a real SKU.
func seedPromotionSKU(ctx context.Context, productRepo *repositories.ProductRepo, variantRepo *repositories.VariantRepo, skuRepo *repositories.SKURepo, skuCode string) (string, error) {
	now := time.Now()
	product := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: "Acai", Slug: "acai-promo-" + skuCode, Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := productRepo.Create(ctx, product); err != nil {
		return "", err
	}
	variant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
	if err := variantRepo.Create(ctx, variant); err != nil {
		return "", err
	}
	sku := &pmsDomain.SKU{TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: skuCode, PriceMinorUnits: 999, CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := skuRepo.Create(ctx, sku); err != nil {
		return "", err
	}
	return sku.ID, nil
}

// TestPromotionRepo_Integration_CreateGetUpdateDeleteRoundTrips proves the
// full lifecycle against real Postgres, including that Delete cascades to
// the promotion's PromotionSKU lines (migration 099's ON DELETE CASCADE).
func TestPromotionRepo_Integration_CreateGetUpdateDeleteRoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	promoRepo := repositories.NewPromotionRepo(db)
	lineRepo := repositories.NewPromotionSKURepo(db)

	var skuID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		skuID, err = seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "ACAI-PROMO-1")
		return err
	})
	require.NoError(t, err)

	var promoID string
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		promo := &promotionsDomain.Promotion{
			TenantID: tenant.IDFromContext(ctx), Name: "Summer Acai", Type: promotionsDomain.PromotionTypePercentage,
			StartsAt: time.Now(), Status: promotionsDomain.PromotionStatusActive,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := promoRepo.Create(ctx, promo); err != nil {
			return err
		}
		promoID = promo.ID
		line := &promotionsDomain.PromotionSKU{
			TenantID: tenant.IDFromContext(ctx), PromotionID: promo.ID, SKUID: skuID,
			DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000, CreatedAt: time.Now(),
		}
		return lineRepo.Create(ctx, line)
	})
	require.NoError(t, err)

	// Update the header.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		promo, err := promoRepo.FindByID(ctx, promoID)
		if err != nil {
			return err
		}
		promo.Name = "Summer Acai v2"
		promo.UpdatedAt = time.Now()
		return promoRepo.Update(ctx, promo)
	})
	require.NoError(t, err)

	var lines []promotionsDomain.PromotionSKU
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		promo, err := promoRepo.FindByID(ctx, promoID)
		if err != nil {
			return err
		}
		require.Equal(t, "Summer Acai v2", promo.Name)
		lines, err = lineRepo.ListByPromotion(ctx, promoID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, skuID, lines[0].SKUID)

	// Delete cascades to the line.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return promoRepo.Delete(ctx, promoID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := promoRepo.FindByID(ctx, promoID)
		return err
	})
	require.ErrorIs(t, err, promotionsDomain.ErrPromotionNotFound)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		lines, err = lineRepo.ListByPromotion(ctx, promoID)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, lines, "deleting the promotion must cascade to its lines")
}

// TestPromotionRepo_Integration_RLSIsolatesTenants proves the new tables'
// tenant_isolation policies actually enforce isolation over the
// non-superuser app role, same reasoning as the other RLS integration
// tests in this file.
func TestPromotionRepo_Integration_RLSIsolatesTenants(t *testing.T) {
	_, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	promoRepo := repositories.NewPromotionRepo(db)

	var promoID string
	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		promo := &promotionsDomain.Promotion{
			TenantID: tenant.IDFromContext(ctx), Name: "Summer Acai", Type: promotionsDomain.PromotionTypePercentage,
			StartsAt: time.Now(), Status: promotionsDomain.PromotionStatusActive,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := promoRepo.Create(ctx, promo); err != nil {
			return err
		}
		promoID = promo.ID
		return nil
	})
	require.NoError(t, err)

	_, ctxB := newTestTenant()
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		_, err := promoRepo.FindByID(ctx, promoID)
		return err
	})
	require.ErrorIs(t, err, promotionsDomain.ErrPromotionNotFound, "tenant B must not see tenant A's promotion")
}
