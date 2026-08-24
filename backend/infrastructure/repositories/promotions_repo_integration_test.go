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

// seedPromotionAndLine returns (promotionID, lineID, error).
func seedPromotionAndLine(ctx context.Context, promoRepo *repositories.PromotionRepo, lineRepo *repositories.PromotionSKURepo, skuID string, status promotionsDomain.PromotionStatus, startsAt time.Time, endsAt *time.Time) (string, string, error) {
	now := time.Now()
	promo := &promotionsDomain.Promotion{
		TenantID: tenant.IDFromContext(ctx), Name: "Promo", Type: promotionsDomain.PromotionTypePercentage,
		StartsAt: startsAt, EndsAt: endsAt, Status: status, CreatedAt: now, UpdatedAt: now,
	}
	if err := promoRepo.Create(ctx, promo); err != nil {
		return "", "", err
	}
	line := &promotionsDomain.PromotionSKU{
		TenantID: tenant.IDFromContext(ctx), PromotionID: promo.ID, SKUID: skuID,
		DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000, CreatedAt: now,
	}
	if err := lineRepo.Create(ctx, line); err != nil {
		return "", "", err
	}
	return promo.ID, line.ID, nil
}

func TestPromotionSKURepo_Integration_ListDueForActivation_ReturnsOnlyNotYetAppliedLinesOfActiveInWindowPromotions(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	promoRepo := repositories.NewPromotionRepo(db)
	lineRepo := repositories.NewPromotionSKURepo(db)

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	var dueLineID, draftLineID, notYetStartedLineID, alreadyEndedLineID, alreadyAppliedLineID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		skuDue, err := seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "PROMO-DUE")
		if err != nil {
			return err
		}
		_, dueLineID, err = seedPromotionAndLine(ctx, promoRepo, lineRepo, skuDue, promotionsDomain.PromotionStatusActive, past, nil)
		if err != nil {
			return err
		}

		skuDraft, err := seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "PROMO-DRAFT")
		if err != nil {
			return err
		}
		_, draftLineID, err = seedPromotionAndLine(ctx, promoRepo, lineRepo, skuDraft, promotionsDomain.PromotionStatusDraft, past, nil)
		if err != nil {
			return err
		}

		skuFuture, err := seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "PROMO-FUTURE")
		if err != nil {
			return err
		}
		_, notYetStartedLineID, err = seedPromotionAndLine(ctx, promoRepo, lineRepo, skuFuture, promotionsDomain.PromotionStatusActive, future, nil)
		if err != nil {
			return err
		}

		skuEnded, err := seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "PROMO-ENDED")
		if err != nil {
			return err
		}
		_, alreadyEndedLineID, err = seedPromotionAndLine(ctx, promoRepo, lineRepo, skuEnded, promotionsDomain.PromotionStatusActive, past, &past)
		if err != nil {
			return err
		}

		skuApplied, err := seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "PROMO-APPLIED")
		if err != nil {
			return err
		}
		_, alreadyAppliedLineID, err = seedPromotionAndLine(ctx, promoRepo, lineRepo, skuApplied, promotionsDomain.PromotionStatusActive, past, nil)
		if err != nil {
			return err
		}
		return lineRepo.MarkApplied(ctx, alreadyAppliedLineID, nil, nil)
	})
	require.NoError(t, err)

	var due []promotionsDomain.PromotionSKU
	err = tenant.WithTenant(tenant.WithSuperAdmin(context.Background()), db, func(ctx context.Context) error {
		var err error
		due, err = lineRepo.ListDueForActivation(ctx, now)
		return err
	})
	require.NoError(t, err)

	ids := make([]string, len(due))
	for i, d := range due {
		ids[i] = d.ID
	}
	require.Contains(t, ids, dueLineID)
	require.NotContains(t, ids, draftLineID)
	require.NotContains(t, ids, notYetStartedLineID)
	require.NotContains(t, ids, alreadyEndedLineID)
	require.NotContains(t, ids, alreadyAppliedLineID)
}

func TestPromotionSKURepo_Integration_ListDueForReversal_ReturnsAppliedLinesWhosePromotionLeftItsWindow(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	promoRepo := repositories.NewPromotionRepo(db)
	lineRepo := repositories.NewPromotionSKURepo(db)

	now := time.Now()
	past := now.Add(-time.Hour)

	var endedAppliedLineID, stillActiveAppliedLineID, endedNotAppliedLineID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		skuEnded, err := seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "REV-ENDED")
		if err != nil {
			return err
		}
		_, endedAppliedLineID, err = seedPromotionAndLine(ctx, promoRepo, lineRepo, skuEnded, promotionsDomain.PromotionStatusEnded, past, nil)
		if err != nil {
			return err
		}
		if err := lineRepo.MarkApplied(ctx, endedAppliedLineID, nil, nil); err != nil {
			return err
		}

		skuActive, err := seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "REV-ACTIVE")
		if err != nil {
			return err
		}
		_, stillActiveAppliedLineID, err = seedPromotionAndLine(ctx, promoRepo, lineRepo, skuActive, promotionsDomain.PromotionStatusActive, past, nil)
		if err != nil {
			return err
		}
		if err := lineRepo.MarkApplied(ctx, stillActiveAppliedLineID, nil, nil); err != nil {
			return err
		}

		skuEndedNotApplied, err := seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "REV-ENDED-NOTAPPLIED")
		if err != nil {
			return err
		}
		_, endedNotAppliedLineID, err = seedPromotionAndLine(ctx, promoRepo, lineRepo, skuEndedNotApplied, promotionsDomain.PromotionStatusEnded, past, nil)
		return err
	})
	require.NoError(t, err)

	var due []promotionsDomain.PromotionSKU
	err = tenant.WithTenant(tenant.WithSuperAdmin(context.Background()), db, func(ctx context.Context) error {
		var err error
		due, err = lineRepo.ListDueForReversal(ctx, now)
		return err
	})
	require.NoError(t, err)

	ids := make([]string, len(due))
	for i, d := range due {
		ids[i] = d.ID
	}
	require.Contains(t, ids, endedAppliedLineID)
	require.NotContains(t, ids, stillActiveAppliedLineID)
	require.NotContains(t, ids, endedNotAppliedLineID)
}

func TestPromotionSKURepo_Integration_MarkAppliedThenMarkReverted_RoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	promoRepo := repositories.NewPromotionRepo(db)
	lineRepo := repositories.NewPromotionSKURepo(db)

	prevPrice := int64(799)
	prevDate := time.Now().AddDate(0, 0, 7).Truncate(24 * time.Hour)
	var promoID, lineID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		skuID, err := seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "ROUNDTRIP-1")
		if err != nil {
			return err
		}
		promoID, lineID, err = seedPromotionAndLine(ctx, promoRepo, lineRepo, skuID, promotionsDomain.PromotionStatusActive, time.Now(), nil)
		return err
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return lineRepo.MarkApplied(ctx, lineID, &prevPrice, &prevDate)
	})
	require.NoError(t, err)

	var afterApply []promotionsDomain.PromotionSKU
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		afterApply, err = lineRepo.ListByPromotion(ctx, promoID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, afterApply, 1)
	require.True(t, afterApply[0].Applied)
	require.NotNil(t, afterApply[0].PreviousSalePriceMinorUnits)
	require.Equal(t, prevPrice, *afterApply[0].PreviousSalePriceMinorUnits)
	require.NotNil(t, afterApply[0].PreviousSaleEndDate)
	require.True(t, afterApply[0].PreviousSaleEndDate.Equal(prevDate))

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return lineRepo.MarkReverted(ctx, lineID)
	})
	require.NoError(t, err)

	var afterRevert []promotionsDomain.PromotionSKU
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		afterRevert, err = lineRepo.ListByPromotion(ctx, promoID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, afterRevert, 1)
	require.False(t, afterRevert[0].Applied)
	require.Nil(t, afterRevert[0].PreviousSalePriceMinorUnits)
	require.Nil(t, afterRevert[0].PreviousSaleEndDate)
}
