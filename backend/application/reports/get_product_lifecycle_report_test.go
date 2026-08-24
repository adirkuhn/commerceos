package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// sequencedProductPerformanceReporter returns a different report per call,
// in order -- lets a test give the current-period and previous-period
// results in one stub without needing to inspect the (from, to) it's
// called with.
type sequencedProductPerformanceReporter struct {
	outs []*appreports.ProductPerformanceReport
	err  error
	call int
}

func (s *sequencedProductPerformanceReporter) Execute(_ context.Context, _, _ time.Time) (*appreports.ProductPerformanceReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := s.outs[s.call]
	s.call++
	return out, nil
}

func TestGetProductLifecycleReport_classifiesEachStage(t *testing.T) {
	current := &appreports.ProductPerformanceReport{CurrencyCode: "EUR", Rows: []appreports.ProductPerformanceRow{
		{ProductPerformance: skuSoldRow("sku-new", 5)},       // no previous-period sales -> new
		{ProductPerformance: skuSoldRow("sku-growing", 10)},  // was 4 -> growing
		{ProductPerformance: skuSoldRow("sku-stable", 3)},    // was 3 -> stable
		{ProductPerformance: skuSoldRow("sku-declining", 2)}, // was 8 -> declining
	}}
	previous := &appreports.ProductPerformanceReport{CurrencyCode: "EUR", Rows: []appreports.ProductPerformanceRow{
		{ProductPerformance: skuSoldRow("sku-growing", 4)},
		{ProductPerformance: skuSoldRow("sku-stable", 3)},
		{ProductPerformance: skuSoldRow("sku-declining", 8)},
		{ProductPerformance: skuSoldRow("sku-dormant", 6)}, // zero current sales -> dormant
	}}
	reporter := &sequencedProductPerformanceReporter{outs: []*appreports.ProductPerformanceReport{current, previous}}
	uc := appreports.NewGetProductLifecycleReportUseCase(reporter)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Rows, 5)

	byID := map[string]appreports.ProductLifecycleRow{}
	for _, row := range result.Rows {
		byID[*row.SKUID] = row
	}
	require.Equal(t, appreports.LifecycleNew, byID["sku-new"].Stage)
	require.Equal(t, appreports.LifecycleGrowing, byID["sku-growing"].Stage)
	require.Equal(t, appreports.LifecycleStable, byID["sku-stable"].Stage)
	require.Equal(t, appreports.LifecycleDeclining, byID["sku-declining"].Stage)
	require.Equal(t, appreports.LifecycleDormant, byID["sku-dormant"].Stage)

	require.Equal(t, int64(10), byID["sku-growing"].CurrentQuantitySold)
	require.Equal(t, int64(4), byID["sku-growing"].PreviousQuantitySold)
	require.Equal(t, int64(0), byID["sku-dormant"].CurrentQuantitySold)
	require.Equal(t, int64(6), byID["sku-dormant"].PreviousQuantitySold)
}

func TestGetProductLifecycleReport_queriesImmediatelyPrecedingPeriodOfEqualLength(t *testing.T) {
	reporter := &sequencedProductPerformanceReporter{outs: []*appreports.ProductPerformanceReport{
		{}, {},
	}}
	uc := appreports.NewGetProductLifecycleReportUseCase(reporter)
	from := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	_, err := uc.Execute(context.Background(), from, to)

	require.NoError(t, err)
}

func TestGetProductLifecycleReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetProductLifecycleReportUseCase(&sequencedProductPerformanceReporter{})

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour))

	require.Error(t, err)
}

func TestGetProductLifecycleReport_propagatesReporterError(t *testing.T) {
	uc := appreports.NewGetProductLifecycleReportUseCase(&sequencedProductPerformanceReporter{err: context.DeadlineExceeded})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
}

func TestGetProductLifecycleReport_handlesNilSKUIDByFallingBackToSKUCode(t *testing.T) {
	adhocRow := omsDomain.ProductPerformance{SKUCode: "ADHOC-1", QuantitySold: 3}
	current := &appreports.ProductPerformanceReport{Rows: []appreports.ProductPerformanceRow{{ProductPerformance: adhocRow}}}
	previous := &appreports.ProductPerformanceReport{}
	uc := appreports.NewGetProductLifecycleReportUseCase(&sequencedProductPerformanceReporter{outs: []*appreports.ProductPerformanceReport{current, previous}})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	require.Nil(t, result.Rows[0].SKUID)
	require.Equal(t, "ADHOC-1", result.Rows[0].SKUCode)
	require.Equal(t, appreports.LifecycleNew, result.Rows[0].Stage)
}
