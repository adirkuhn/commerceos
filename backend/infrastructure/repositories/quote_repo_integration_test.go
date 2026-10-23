//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/quotes"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func newQuote(t *testing.T, tenantID, name string, lines ...quotes.Line) *quotes.Quote {
	t.Helper()
	q, err := quotes.NewQuote(tenantID, quotes.Input{Name: name, DiscountBasisPoints: 3000, CurrencyCode: "EUR", Lines: lines},
		"user-1", time.Now().UTC().Truncate(time.Microsecond))
	require.NoError(t, err)
	return q
}

func quoteLine(qty, price int64) quotes.Line {
	return quotes.Line{ProductID: idgen.NewV4(), SKUID: idgen.NewV4(), Quantity: qty, UnitPriceMinorUnits: price}
}

// Runs as commerceos_app so RLS applies, like production.
func TestQuoteRepo_Integration_CRUDNumberingAndIsolation(t *testing.T) {
	_, dsn := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, dsn)
	repo := repositories.NewQuoteRepo(db)
	tenantA, ctxA := newTestTenant()
	tenantB, ctxB := newTestTenant()
	inA := func(fn func(ctx context.Context) error) { require.NoError(t, tenant.WithTenant(ctxA, db, fn)) }

	first := newQuote(t, tenantA, "Restaurant Sabor", quoteLine(100, 871), quoteLine(5, 1250))
	second := newQuote(t, tenantA, "Mercado Central")
	other := newQuote(t, tenantB, "Other shop")
	inA(func(ctx context.Context) error {
		require.NoError(t, repo.Create(ctx, first))
		return repo.Create(ctx, second)
	})
	require.NoError(t, tenant.WithTenant(ctxB, db, func(ctx context.Context) error { return repo.Create(ctx, other) }))
	assert.Equal(t, int64(1), first.Number)
	assert.Equal(t, int64(2), second.Number)
	assert.Equal(t, int64(1), other.Number, "numbering is per shop")

	// Read back with lines in order.
	inA(func(ctx context.Context) error {
		got, err := repo.FindByID(ctx, first.ID)
		require.NoError(t, err)
		assert.Equal(t, first.Lines, got.Lines)
		assert.Equal(t, "Restaurant Sabor", got.Name)
		assert.Equal(t, 3000, got.DiscountBasisPoints)
		assert.True(t, first.CreatedAt.Equal(got.CreatedAt))
		return nil
	})

	// Saving over replaces the lines.
	replacement := quoteLine(2, 500)
	inA(func(ctx context.Context) error {
		require.NoError(t, first.Revise(quotes.Input{Name: "Sabor (revised)", CurrencyCode: "EUR", Lines: []quotes.Line{replacement}}, time.Now().UTC()))
		require.NoError(t, repo.Update(ctx, first))
		got, err := repo.FindByID(ctx, first.ID)
		require.NoError(t, err)
		assert.Equal(t, []quotes.Line{replacement}, got.Lines)
		assert.Equal(t, "Sabor (revised)", got.Name)
		return nil
	})

	// List: newest save first, totals per quote, search by name or number.
	inA(func(ctx context.Context) error {
		list, total, err := repo.List(ctx, quotes.ListFilter{Limit: 50})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total, "the other shop's quote is invisible")
		require.Len(t, list, 2)
		assert.Equal(t, first.ID, list[0].ID)
		assert.Equal(t, 1, list[0].LineCount)
		assert.Equal(t, int64(1000), list[0].TotalMinorUnits)
		assert.Equal(t, 0, list[1].LineCount)

		byName, _, err := repo.List(ctx, quotes.ListFilter{Search: "mercado", Limit: 50})
		require.NoError(t, err)
		require.Len(t, byName, 1)
		assert.Equal(t, second.ID, byName[0].ID)

		byNumber, _, err := repo.List(ctx, quotes.ListFilter{Search: "Q-0002", Limit: 50})
		require.NoError(t, err)
		require.Len(t, byNumber, 1)
		assert.Equal(t, second.ID, byNumber[0].ID)

		paged, total, err := repo.List(ctx, quotes.ListFilter{Limit: 1, Offset: 1})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		require.Len(t, paged, 1)
		assert.Equal(t, second.ID, paged[0].ID)
		return nil
	})

	// Another shop can neither read nor delete it.
	require.NoError(t, tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, first.ID)
		assert.ErrorIs(t, err, quotes.ErrQuoteNotFound)
		assert.ErrorIs(t, repo.Delete(ctx, first.ID), quotes.ErrQuoteNotFound)
		return nil
	}))

	// Deleting the newest quote doesn't hand its number out again.
	third := newQuote(t, tenantA, "After delete")
	inA(func(ctx context.Context) error {
		require.NoError(t, repo.Delete(ctx, second.ID))
		_, err := repo.FindByID(ctx, second.ID)
		assert.ErrorIs(t, err, quotes.ErrQuoteNotFound)
		assert.ErrorIs(t, repo.Delete(ctx, second.ID), quotes.ErrQuoteNotFound)
		return repo.Create(ctx, third)
	})
	assert.Equal(t, int64(3), third.Number)
}

func TestQuoteRepo_Integration_UpdateMissingIsNotFound(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewQuoteRepo(db)
	q := newQuote(t, tenantID, "Ghost")
	q.ID = idgen.NewV4()

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		assert.ErrorIs(t, repo.Update(ctx, q), quotes.ErrQuoteNotFound)
		return nil
	}))
}
