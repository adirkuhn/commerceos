package reports

import (
	"context"
	"fmt"
	"sort"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// RFMSegment is a customer's rule-based classification from their
// Recency/Frequency/Monetary quintile scores -- the standard RFM
// segmentation scheme (Phase 1 "Sales Intelligence" roadmap item).
type RFMSegment string

const (
	// RFMSegmentChampions bought recently, often, and spent the most.
	RFMSegmentChampions RFMSegment = "champions"
	// RFMSegmentLoyalCustomers buy often and recently, though not
	// necessarily the highest spenders.
	RFMSegmentLoyalCustomers RFMSegment = "loyal_customers"
	// RFMSegmentPotentialLoyalists bought recently but haven't ordered
	// often yet -- newer customers worth nurturing.
	RFMSegmentPotentialLoyalists RFMSegment = "potential_loyalists"
	// RFMSegmentAtRisk used to order often but haven't recently.
	RFMSegmentAtRisk RFMSegment = "at_risk"
	// RFMSegmentLost neither recent nor frequent.
	RFMSegmentLost RFMSegment = "lost"
	// RFMSegmentNeedsAttention is everyone else -- mid-range on every axis.
	RFMSegmentNeedsAttention RFMSegment = "needs_attention"
)

// CustomerRFMRow is one customer's Recency/Frequency/Monetary quintile
// scores (1-5, 5 = best) and resulting segment, alongside the raw figures
// they were computed from.
type CustomerRFMRow struct {
	omsDomain.CustomerPerformance
	RecencyScore   int64
	FrequencyScore int64
	MonetaryScore  int64
	Segment        RFMSegment
}

// CustomerRFMReport is the Phase 1 "Sales Intelligence" roadmap's RFM
// segmentation report: every customer with a qualifying order in
// [From, To) (same set as the Customer Performance report), scored and
// segmented relative to the other customers in this same result set --
// quintiles are always relative to the tenant's own customer base for the
// period, not a fixed threshold (same reasoning as the Product Performance
// Matrix's medians).
type CustomerRFMReport struct {
	CurrencyCode string
	Rows         []CustomerRFMRow
}

// GetCustomerRFMReportUseCase is pure composition over the Customer
// Performance report -- no new query, same pattern as
// GetProductPerformanceMatrixUseCase.
type GetCustomerRFMReportUseCase struct {
	customerPerformance customerPerformanceReporter
}

// customerPerformanceReporter narrows GetCustomerPerformanceReportUseCase
// to the one method this report needs (ISP).
type customerPerformanceReporter interface {
	Execute(ctx context.Context, from, to time.Time) (*CustomerPerformanceReport, error)
}

func NewGetCustomerRFMReportUseCase(customerPerformance customerPerformanceReporter) *GetCustomerRFMReportUseCase {
	return &GetCustomerRFMReportUseCase{customerPerformance: customerPerformance}
}

func (uc *GetCustomerRFMReportUseCase) Execute(ctx context.Context, from, to time.Time) (*CustomerRFMReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getCustomerRFMReport: %w", ErrToBeforeFrom)
	}

	perf, err := uc.customerPerformance.Execute(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("getCustomerRFMReport: %w", err)
	}

	n := len(perf.Rows)
	recencyValues := make([]int64, n)
	frequencyValues := make([]int64, n)
	monetaryValues := make([]int64, n)
	for i, row := range perf.Rows {
		// More recent = better = higher score, so rank by negative
		// duration-since-last-order: the smallest (least negative)
		// duration -- i.e. the most recent order -- sorts last, same
		// direction as frequency/monetary where "biggest number wins".
		recencyValues[i] = -int64(to.Sub(row.LastOrderAt))
		frequencyValues[i] = row.OrderCount
		monetaryValues[i] = row.RevenueMinorUnits
	}
	recencyScores := quintileScores(recencyValues)
	frequencyScores := quintileScores(frequencyValues)
	monetaryScores := quintileScores(monetaryValues)

	report := &CustomerRFMReport{CurrencyCode: perf.CurrencyCode}
	for i, row := range perf.Rows {
		r, f, m := recencyScores[i], frequencyScores[i], monetaryScores[i]
		report.Rows = append(report.Rows, CustomerRFMRow{
			CustomerPerformance: row,
			RecencyScore:        r, FrequencyScore: f, MonetaryScore: m,
			Segment: classifyRFMSegment(r, f, m),
		})
	}
	return report, nil
}

// quintileScores ranks values ascending and buckets them into 5 equal-ish
// groups (1 = lowest, 5 = highest) -- ties keep their original relative
// order (Go's sort.Slice is not stable, so equal values can land in
// either neighboring bucket; acceptable for a v1 relative-ranking report).
func quintileScores(values []int64) []int64 {
	n := len(values)
	scores := make([]int64, n)
	if n == 0 {
		return scores
	}
	type ranked struct {
		index int
		value int64
	}
	sorted := make([]ranked, n)
	for i, v := range values {
		sorted[i] = ranked{index: i, value: v}
	}
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].value < sorted[j].value })
	for rank, r := range sorted {
		bucket := int64(rank*5/n) + 1
		if bucket > 5 {
			bucket = 5
		}
		scores[r.index] = bucket
	}
	return scores
}

// classifyRFMSegment is a fixed, standard RFM rule table -- see
// RFMSegment's doc for what each label means.
func classifyRFMSegment(r, f, m int64) RFMSegment {
	switch {
	case r >= 4 && f >= 4 && m >= 4:
		return RFMSegmentChampions
	case r >= 3 && f >= 4:
		return RFMSegmentLoyalCustomers
	case r >= 4 && f <= 3:
		return RFMSegmentPotentialLoyalists
	case r <= 2 && f >= 3:
		return RFMSegmentAtRisk
	case r <= 2 && f <= 2:
		return RFMSegmentLost
	default:
		return RFMSegmentNeedsAttention
	}
}
