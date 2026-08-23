package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type stubProductPerformanceReporter struct {
	out     *appreports.ProductPerformanceReport
	err     error
	gotFrom time.Time
	gotTo   time.Time
}

func (s *stubProductPerformanceReporter) Execute(_ context.Context, from, to time.Time) (*appreports.ProductPerformanceReport, error) {
	s.gotFrom, s.gotTo = from, to
	return s.out, s.err
}

func perfRow(skuCode string, revenue, marginBP int64) appreports.ProductPerformanceRow {
	margin := marginBP
	return appreports.ProductPerformanceRow{
		ProductPerformance: omsDomain.ProductPerformance{SKUCode: skuCode, RevenueMinorUnits: revenue},
		MarginBasisPoints:  &margin,
	}
}

func TestGetProductPerformanceMatrix_classifiesFourQuadrantsAroundMedians(t *testing.T) {
	reporter := &stubProductPerformanceReporter{out: &appreports.ProductPerformanceReport{
		CurrencyCode: "EUR",
		Rows: []appreports.ProductPerformanceRow{
			perfRow("WINNER", 10000, 5000),   // high revenue, high margin
			perfRow("STAR", 9000, 500),       // high revenue, low margin
			perfRow("HIDDEN-GEM", 100, 6000), // low revenue, high margin
			perfRow("PROBLEM", 50, 400),      // low revenue, low margin
		},
	}}
	uc := appreports.NewGetProductPerformanceMatrixUseCase(reporter)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Equal(t, "EUR", result.CurrencyCode)
	byCode := map[string]appreports.ProductPerformanceQuadrant{}
	for _, row := range result.Rows {
		byCode[row.SKUCode] = row.Quadrant
	}
	require.Equal(t, appreports.QuadrantWinners, byCode["WINNER"])
	require.Equal(t, appreports.QuadrantStars, byCode["STAR"])
	require.Equal(t, appreports.QuadrantHiddenGems, byCode["HIDDEN-GEM"])
	require.Equal(t, appreports.QuadrantProblems, byCode["PROBLEM"])
}

func TestGetProductPerformanceMatrix_excludesRowsWithUnknownMargin(t *testing.T) {
	reporter := &stubProductPerformanceReporter{out: &appreports.ProductPerformanceReport{
		Rows: []appreports.ProductPerformanceRow{
			perfRow("KNOWN", 100, 1000),
			{ProductPerformance: omsDomain.ProductPerformance{SKUCode: "NO-MARGIN", RevenueMinorUnits: 200}, MarginBasisPoints: nil},
		},
	}}
	uc := appreports.NewGetProductPerformanceMatrixUseCase(reporter)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	require.Equal(t, "KNOWN", result.Rows[0].SKUCode)
}

func TestGetProductPerformanceMatrix_propagatesUnderlyingError(t *testing.T) {
	reporter := &stubProductPerformanceReporter{err: context.DeadlineExceeded}
	uc := appreports.NewGetProductPerformanceMatrixUseCase(reporter)

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
}

func TestGetProductPerformanceMatrix_emptyInputProducesEmptyMatrix(t *testing.T) {
	reporter := &stubProductPerformanceReporter{out: &appreports.ProductPerformanceReport{CurrencyCode: "USD"}}
	uc := appreports.NewGetProductPerformanceMatrixUseCase(reporter)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Empty(t, result.Rows)
}
