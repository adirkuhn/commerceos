package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type stubCustomerCohortReportRepo struct {
	sizes     []omsDomain.CohortSize
	retention []omsDomain.CohortRetentionPoint
	sizesErr  error
	retainErr error
}

func (s *stubCustomerCohortReportRepo) CohortSizesBetween(_ context.Context, _, _ time.Time, _ string) ([]omsDomain.CohortSize, error) {
	if s.sizesErr != nil {
		return nil, s.sizesErr
	}
	return s.sizes, nil
}

func (s *stubCustomerCohortReportRepo) CohortRetentionBetween(_ context.Context, _, _ time.Time, _ string) ([]omsDomain.CohortRetentionPoint, error) {
	if s.retainErr != nil {
		return nil, s.retainErr
	}
	return s.retention, nil
}

func TestGetCustomerCohortsReport_groupsRetentionUnderItsCohortWithRate(t *testing.T) {
	cohortMonth := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := &stubCustomerCohortReportRepo{
		sizes: []omsDomain.CohortSize{{CohortMonth: cohortMonth, CustomerCount: 4}},
		retention: []omsDomain.CohortRetentionPoint{
			{CohortMonth: cohortMonth, MonthsSinceCohort: 0, RetainedCustomers: 4, RevenueMinorUnits: 4000},
			{CohortMonth: cohortMonth, MonthsSinceCohort: 1, RetainedCustomers: 1, RevenueMinorUnits: 500},
		},
	}
	uc := appreports.NewGetCustomerCohortsReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Cohorts, 1)
	cohort := result.Cohorts[0]
	require.True(t, cohort.CohortMonth.Equal(cohortMonth))
	require.Equal(t, int64(4), cohort.CohortSize)
	require.Len(t, cohort.Retention, 2)
	require.Equal(t, int64(0), cohort.Retention[0].MonthsSinceCohort)
	require.Equal(t, int64(4), cohort.Retention[0].RetainedCustomers)
	require.Equal(t, int64(10000), cohort.Retention[0].RetentionRateBasisPoints) // 4/4 = 100%
	require.Equal(t, int64(1), cohort.Retention[1].MonthsSinceCohort)
	require.Equal(t, int64(1), cohort.Retention[1].RetainedCustomers)
	require.Equal(t, int64(2500), cohort.Retention[1].RetentionRateBasisPoints) // 1/4 = 25%
}

func TestGetCustomerCohortsReport_zeroCohortSizeGivesZeroRate(t *testing.T) {
	cohortMonth := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := &stubCustomerCohortReportRepo{
		sizes:     []omsDomain.CohortSize{{CohortMonth: cohortMonth, CustomerCount: 0}},
		retention: nil,
	}
	uc := appreports.NewGetCustomerCohortsReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Cohorts, 1)
	require.Empty(t, result.Cohorts[0].Retention)
}

func TestGetCustomerCohortsReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetCustomerCohortsReportUseCase(newMockGormDB(t), &stubCustomerCohortReportRepo{}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour))

	require.Error(t, err)
}

func TestGetCustomerCohortsReport_propagatesRepoError(t *testing.T) {
	uc := appreports.NewGetCustomerCohortsReportUseCase(newMockGormDB(t), &stubCustomerCohortReportRepo{sizesErr: context.DeadlineExceeded}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
}
