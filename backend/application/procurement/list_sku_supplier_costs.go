package procurement

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// supplierFinderForCosts names the suppliers in a cost history (ISP).
type supplierFinderForCosts interface {
	FindByID(ctx context.Context, id string) (*suppliersDomain.Supplier, error)
}

// ListSKUSupplierCostsInput: one page of a SKU's cost history, optionally
// one supplier's. Cursor is an offset.
type ListSKUSupplierCostsInput struct {
	SKUID      string
	SupplierID string
	Cursor     int
	Limit      int
}

// SKUSupplierCosts is what the product cost panel shows: the summary over
// the whole history, one page of its rows, and the names of the suppliers
// in it (a deleted supplier has none).
type SKUSupplierCosts struct {
	Currency      string
	Summary       procurementDomain.SupplierCostSummary
	Page          procurementDomain.SupplierCostPage
	SupplierNames map[string]string
}

// ListSKUSupplierCostsUseCase reads a SKU's cost history by supplier for
// GET /pms/skus/{id}/supplier-costs (COM-11).
//
//events:none read-only
type ListSKUSupplierCostsUseCase struct {
	db        *gorm.DB
	costs     procurementDomain.SupplierCostRepository
	settings  tenantSettingsReader
	suppliers supplierFinderForCosts
}

func NewListSKUSupplierCostsUseCase(db *gorm.DB, costs procurementDomain.SupplierCostRepository, settings tenantSettingsReader,
	suppliers supplierFinderForCosts) *ListSKUSupplierCostsUseCase {
	return &ListSKUSupplierCostsUseCase{db: db, costs: costs, settings: settings, suppliers: suppliers}
}

func (uc *ListSKUSupplierCostsUseCase) Execute(ctx context.Context, in ListSKUSupplierCostsInput) (SKUSupplierCosts, error) {
	var out SKUSupplierCosts
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.settings.Get(ctx)
		if err != nil {
			return err
		}
		entries, err := uc.costs.ListBySKU(ctx, in.SKUID)
		if err != nil {
			return err
		}
		out.Currency = s.Currency()
		out.Summary = procurementDomain.SummarizeSupplierCosts(entries, out.Currency)
		out.Page = procurementDomain.PageSupplierCosts(entries, in.SupplierID, in.Cursor, in.Limit)
		out.SupplierNames, err = uc.names(ctx, entries)
		return err
	})
	if err != nil {
		return SKUSupplierCosts{}, fmt.Errorf("listSKUSupplierCosts: %w", err)
	}
	return out, nil
}

func (uc *ListSKUSupplierCostsUseCase) names(ctx context.Context, entries []procurementDomain.SupplierCostEntry) (map[string]string, error) {
	names := map[string]string{}
	seen := map[string]bool{}
	for _, e := range entries {
		if e.SupplierID == nil || seen[*e.SupplierID] {
			continue
		}
		seen[*e.SupplierID] = true
		sup, err := uc.suppliers.FindByID(ctx, *e.SupplierID)
		if errors.Is(err, suppliersDomain.ErrSupplierNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		names[sup.ID] = sup.Name
	}
	return names, nil
}
