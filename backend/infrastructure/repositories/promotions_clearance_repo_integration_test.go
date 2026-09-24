//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// seedClearanceLine adds a clearance line (BatchID set) for skuID to an
// existing promotion. batch_id has no FK (see migration 183), so any UUID
// stands in for a real batch here.
func seedClearanceLine(ctx context.Context, lineRepo *repositories.PromotionSKURepo, promotionID, skuID, batchID string) (string, error) {
	line := &promotionsDomain.PromotionSKU{
		TenantID: tenant.IDFromContext(ctx), PromotionID: promotionID, SKUID: skuID, BatchID: &batchID,
		DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 3000, CreatedAt: time.Now(),
	}
	if err := lineRepo.Create(ctx, line); err != nil {
		return "", err
	}
	return line.ID, nil
}

func lineIDs(lines []promotionsDomain.PromotionSKU) []string {
	ids := make([]string, len(lines))
	for i, l := range lines {
		ids[i] = l.ID
	}
	return ids
}

// TestPromotionSKURepo_Integration_ClearanceLifecycle proves a clearance
// line round-trips its BatchID, shows up as open until MarkFinished, and
// that a finished line is never due for activation but is due for reversal
// while still applied -- even though its promotion is still in its window.
func TestPromotionSKURepo_Integration_ClearanceLifecycle(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	promoRepo := repositories.NewPromotionRepo(db)
	lineRepo := repositories.NewPromotionSKURepo(db)

	now := time.Now()
	past := now.Add(-time.Hour)
	batchID := idgen.NewV4()

	var promoID, ordinaryLineID, clearanceLineID, skuID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		skuID, err = seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "CLEAR-1")
		if err != nil {
			return err
		}
		otherSKU, err := seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "CLEAR-ORD")
		if err != nil {
			return err
		}
		promoID, ordinaryLineID, err = seedPromotionAndLine(ctx, promoRepo, lineRepo, otherSKU, promotionsDomain.PromotionStatusActive, past, nil)
		if err != nil {
			return err
		}
		clearanceLineID, err = seedClearanceLine(ctx, lineRepo, promoID, skuID, batchID)
		return err
	})
	require.NoError(t, err)

	// Round-trip + open listings.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		lines, err := lineRepo.ListByPromotion(ctx, promoID)
		require.NoError(t, err)
		for _, l := range lines {
			if l.ID == clearanceLineID {
				require.NotNil(t, l.BatchID)
				require.Equal(t, batchID, *l.BatchID)
				require.Nil(t, l.FinishedAt)
			} else {
				require.Nil(t, l.BatchID)
			}
		}

		open, err := lineRepo.ListOpenBySKU(ctx, skuID)
		require.NoError(t, err)
		require.Equal(t, []string{clearanceLineID}, lineIDs(open))

		unfinished, err := lineRepo.CountUnfinished(ctx, promoID)
		require.NoError(t, err)
		require.Equal(t, int64(2), unfinished)

		return lineRepo.MarkApplied(ctx, clearanceLineID, nil, nil)
	})
	require.NoError(t, err)

	adminCtx := tenant.WithSuperAdmin(context.Background())
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		open, err := lineRepo.ListOpen(ctx)
		require.NoError(t, err)
		require.Contains(t, lineIDs(open), clearanceLineID)
		require.NotContains(t, lineIDs(open), ordinaryLineID)

		reversal, err := lineRepo.ListDueForReversal(ctx, now)
		require.NoError(t, err)
		require.NotContains(t, lineIDs(reversal), clearanceLineID)
		return nil
	})
	require.NoError(t, err)

	// Finish it.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		require.NoError(t, lineRepo.MarkFinished(ctx, clearanceLineID, now))

		open, err := lineRepo.ListOpenBySKU(ctx, skuID)
		require.NoError(t, err)
		require.Empty(t, open)

		unfinished, err := lineRepo.CountUnfinished(ctx, promoID)
		require.NoError(t, err)
		require.Equal(t, int64(1), unfinished) // the ordinary line never finishes
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		open, err := lineRepo.ListOpen(ctx)
		require.NoError(t, err)
		require.NotContains(t, lineIDs(open), clearanceLineID)

		reversal, err := lineRepo.ListDueForReversal(ctx, now)
		require.NoError(t, err)
		require.Contains(t, lineIDs(reversal), clearanceLineID, "finished+applied line is due for reversal even inside its promotion's window")

		require.NoError(t, lineRepo.MarkReverted(ctx, clearanceLineID))
		activation, err := lineRepo.ListDueForActivation(ctx, now)
		require.NoError(t, err)
		require.NotContains(t, lineIDs(activation), clearanceLineID, "finished line is never re-applied")
		return nil
	})
	require.NoError(t, err)
}
