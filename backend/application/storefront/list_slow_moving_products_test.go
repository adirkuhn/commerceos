package storefront_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

type stubSlowMovingReport struct {
	out     *appreports.SlowMovingProductsReport
	err     error
	gotFrom time.Time
	gotTo   time.Time
	calls   int
}

func (s *stubSlowMovingReport) Execute(_ context.Context, from, to time.Time) (*appreports.SlowMovingProductsReport, error) {
	s.calls++
	s.gotFrom, s.gotTo = from, to
	return s.out, s.err
}

var slowMovingNow = time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)

func newSlowMovingUC(report *stubSlowMovingReport) *appstorefront.ListSlowMovingProductsUseCase {
	return appstorefront.NewListSlowMovingProductsUseCase(report, func() time.Time { return slowMovingNow })
}

func TestListSlowMovingProducts_queriesTrailingWindowEndingNow(t *testing.T) {
	report := &stubSlowMovingReport{out: &appreports.SlowMovingProductsReport{}}

	_, err := newSlowMovingUC(report).Execute(context.Background(), 60, 12)

	require.NoError(t, err)
	assert.Equal(t, slowMovingNow, report.gotTo)
	assert.Equal(t, slowMovingNow.AddDate(0, 0, -60), report.gotFrom)
}

func TestListSlowMovingProducts_sortsByStockOnHandDescAndCapsToLimit(t *testing.T) {
	report := &stubSlowMovingReport{out: &appreports.SlowMovingProductsReport{Rows: []appreports.SlowMovingProductRow{
		{SKUCode: "LOW", ProductName: "Low", StockOnHand: 2},
		{SKUCode: "HIGH", ProductName: "High", StockOnHand: 40},
		{SKUCode: "MID", ProductName: "Mid", StockOnHand: 10},
	}}}

	got, err := newSlowMovingUC(report).Execute(context.Background(), 30, 2)

	require.NoError(t, err)
	assert.Equal(t, 30, got.WindowDays)
	require.Len(t, got.Products, 2)
	assert.Equal(t, appstorefront.SlowMovingProduct{SKUCode: "HIGH", ProductName: "High", StockOnHand: 40}, got.Products[0])
	assert.Equal(t, "MID", got.Products[1].SKUCode)
}

func TestListSlowMovingProducts_emptyReportReturnsEmptyList(t *testing.T) {
	report := &stubSlowMovingReport{out: &appreports.SlowMovingProductsReport{}}

	got, err := newSlowMovingUC(report).Execute(context.Background(), 60, 12)

	require.NoError(t, err)
	assert.NotNil(t, got.Products)
	assert.Empty(t, got.Products)
}

func TestListSlowMovingProducts_rejectsOutOfRangeWindow(t *testing.T) {
	for _, days := range []int{0, -1, 366} {
		report := &stubSlowMovingReport{}

		_, err := newSlowMovingUC(report).Execute(context.Background(), days, 12)

		assert.ErrorIs(t, err, storefrontDomain.ErrInvalidSlowMovingWindow, "days=%d", days)
		assert.Zero(t, report.calls, "days=%d must not hit the report", days)
	}
}

func TestListSlowMovingProducts_rejectsOutOfRangeLimit(t *testing.T) {
	for _, limit := range []int{0, 51} {
		report := &stubSlowMovingReport{}

		_, err := newSlowMovingUC(report).Execute(context.Background(), 60, limit)

		assert.ErrorIs(t, err, storefrontDomain.ErrInvalidSlowMovingLimit, "limit=%d", limit)
		assert.Zero(t, report.calls)
	}
}

func TestListSlowMovingProducts_wrapsReportError(t *testing.T) {
	boom := errors.New("boom")
	report := &stubSlowMovingReport{err: boom}

	_, err := newSlowMovingUC(report).Execute(context.Background(), 60, 12)

	assert.ErrorIs(t, err, boom)
}
