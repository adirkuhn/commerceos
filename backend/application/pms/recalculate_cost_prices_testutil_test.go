package pms_test

import (
	"context"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

type stubBatchListerForCostPrice struct {
	batches []invDomain.Batch
	err     error
}

func (s *stubBatchListerForCostPrice) Execute(_ context.Context, _ string) ([]invDomain.Batch, error) {
	return s.batches, s.err
}

type stubPurchaseCostLineListerForCostPrice struct {
	lines []apppms.PurchaseCostLine
	err   error
}

func (s *stubPurchaseCostLineListerForCostPrice) Execute(_ context.Context, _ string) ([]apppms.PurchaseCostLine, error) {
	return s.lines, s.err
}

type stubSKUCostPriceHistoryRepo struct {
	upserted   []pmsDomain.SKUCostPriceHistory
	upsertErr  error
	listResult []pmsDomain.SKUCostPriceHistory
	listErr    error
}

func (s *stubSKUCostPriceHistoryRepo) Upsert(_ context.Context, h *pmsDomain.SKUCostPriceHistory) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	h.ID = "history-1"
	s.upserted = append(s.upserted, *h)
	return nil
}
func (s *stubSKUCostPriceHistoryRepo) ListBySKU(_ context.Context, _ string, _ int) ([]pmsDomain.SKUCostPriceHistory, error) {
	return s.listResult, s.listErr
}

type stubSettingsRepoForPMS struct {
	settings *settingsDomain.TenantSettings
	err      error
}

func (s *stubSettingsRepoForPMS) Get(_ context.Context) (*settingsDomain.TenantSettings, error) {
	if s.settings == nil && s.err == nil {
		return &settingsDomain.TenantSettings{}, nil
	}
	return s.settings, s.err
}
func (s *stubSettingsRepoForPMS) Upsert(_ context.Context, _ *settingsDomain.TenantSettings) error {
	return nil
}