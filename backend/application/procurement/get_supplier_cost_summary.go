package procurement

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// tenantSettingsReader gives the tenant's currency (ISP).
type tenantSettingsReader interface {
	Get(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

// SupplierCostReport is a SKU's cost history by supplier: every row, and
// the summary (last cost, averages) in the tenant's currency.
type SupplierCostReport struct {
	Currency string
	Entries  []procurementDomain.SupplierCostEntry
	Summary  procurementDomain.SupplierCostSummary
}

// GetSupplierCostSummaryUseCase reads a SKU's cost history by supplier and
// summarizes it (procurementDomain.SummarizeSupplierCosts) in the tenant's
// currency -- rows in another currency are kept in Entries but left out of
// the summary, and counted.
//
//events:none read-only
type GetSupplierCostSummaryUseCase struct {
	db       *gorm.DB
	costs    procurementDomain.SupplierCostRepository
	settings tenantSettingsReader
}

func NewGetSupplierCostSummaryUseCase(db *gorm.DB, costs procurementDomain.SupplierCostRepository, settings tenantSettingsReader) *GetSupplierCostSummaryUseCase {
	return &GetSupplierCostSummaryUseCase{db: db, costs: costs, settings: settings}
}

func (uc *GetSupplierCostSummaryUseCase) Execute(ctx context.Context, skuID string) (SupplierCostReport, error) {
	var report SupplierCostReport
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.settings.Get(ctx)
		if err != nil {
			return err
		}
		entries, err := uc.costs.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		report = SupplierCostReport{Currency: s.Currency(), Entries: entries}
		report.Summary = procurementDomain.SummarizeSupplierCosts(entries, report.Currency)
		return nil
	})
	if err != nil {
		return SupplierCostReport{}, fmt.Errorf("getSupplierCostSummary: %w", err)
	}
	return report, nil
}
