package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type stubCustomerPerformanceReporter struct {
	out *appreports.CustomerPerformanceReport
	err error
}

func (s *stubCustomerPerformanceReporter) Execute(_ context.Context, _, _ time.Time) (*appreports.CustomerPerformanceReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

func TestGetCustomerRFMReport_scoresAndSegmentsCustomersAcrossTheirRelativeRanking(t *testing.T) {
	to := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	from := to.AddDate(0, -1, 0)
	// 5 customers so quintile buckets land on clean boundaries (1..5).
	rows := []omsDomain.CustomerPerformance{
		{CustomerID: "c1", CustomerName: "Least Recent/Frequent/Spend", OrderCount: 1, RevenueMinorUnits: 100, LastOrderAt: to.AddDate(0, 0, -30)},
		{CustomerID: "c2", OrderCount: 2, RevenueMinorUnits: 200, LastOrderAt: to.AddDate(0, 0, -20)},
		{CustomerID: "c3", OrderCount: 3, RevenueMinorUnits: 300, LastOrderAt: to.AddDate(0, 0, -10)},
		{CustomerID: "c4", OrderCount: 4, RevenueMinorUnits: 400, LastOrderAt: to.AddDate(0, 0, -5)},
		{CustomerID: "c5", CustomerName: "Most Recent/Frequent/Spend", OrderCount: 5, RevenueMinorUnits: 500, LastOrderAt: to.AddDate(0, 0, -1)},
	}
	reporter := &stubCustomerPerformanceReporter{out: &appreports.CustomerPerformanceReport{CurrencyCode: "EUR", Rows: rows}}
	uc := appreports.NewGetCustomerRFMReportUseCase(reporter)

	result, err := uc.Execute(context.Background(), from, to)

	require.NoError(t, err)
	require.Equal(t, "EUR", result.CurrencyCode)
	require.Len(t, result.Rows, 5)

	byID := map[string]appreports.CustomerRFMRow{}
	for _, r := range result.Rows {
		byID[r.CustomerID] = r
	}

	// c1: least recent, least frequent, lowest spend -> bottom of every axis.
	require.Equal(t, int64(1), byID["c1"].RecencyScore)
	require.Equal(t, int64(1), byID["c1"].FrequencyScore)
	require.Equal(t, int64(1), byID["c1"].MonetaryScore)
	require.Equal(t, appreports.RFMSegmentLost, byID["c1"].Segment)

	// c5: most recent, most frequent, highest spend -> top of every axis.
	require.Equal(t, int64(5), byID["c5"].RecencyScore)
	require.Equal(t, int64(5), byID["c5"].FrequencyScore)
	require.Equal(t, int64(5), byID["c5"].MonetaryScore)
	require.Equal(t, appreports.RFMSegmentChampions, byID["c5"].Segment)
}

func TestGetCustomerRFMReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetCustomerRFMReportUseCase(&stubCustomerPerformanceReporter{out: &appreports.CustomerPerformanceReport{}})

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour))

	require.Error(t, err)
}

func TestGetCustomerRFMReport_propagatesReporterError(t *testing.T) {
	uc := appreports.NewGetCustomerRFMReportUseCase(&stubCustomerPerformanceReporter{err: context.DeadlineExceeded})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
}

func TestGetCustomerRFMReport_emptyCustomerListGivesEmptyRows(t *testing.T) {
	uc := appreports.NewGetCustomerRFMReportUseCase(&stubCustomerPerformanceReporter{out: &appreports.CustomerPerformanceReport{CurrencyCode: "EUR"}})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Empty(t, result.Rows)
}
