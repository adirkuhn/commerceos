package procurement

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/domain/errs"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/domain/procurement/planning"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

var (
	ErrInvalidPlanOrder = errs.New("PLAN_ORDER_INVALID", "a planned order needs at least one line, each SKU once with a quantity above 0")
	ErrSKUNotPlannable  = errs.New("SKU_NOT_PLANNABLE", "SKU can't be bought: unknown, inactive, or a weight pool's package")
)

type PlanOrderLine struct {
	SKUID string
	// Quantity in purchased items (boxes, kg) -- as the plan shows it.
	Quantity int64
}

type PlanOrderInput struct {
	SupplierID string
	Lines      []PlanOrderLine
	Notes      string
}

type planSupplierFinder interface {
	FindByID(ctx context.Context, id string) (*suppliersDomain.Supplier, error)
}

type planTenantSettings interface {
	Get(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

type planPurchaseOrderCreator interface {
	Create(ctx context.Context, po *procurementDomain.PurchaseOrder) error
}

type planLineItemCreator interface {
	Create(ctx context.Context, li *procurementDomain.PurchaseOrderLineItem) error
}

type PlanOrderDeps struct {
	Catalog        planning.CatalogReader
	Supply         planning.SupplyReader
	Suppliers      planSupplierFinder
	Settings       planTenantSettings
	PurchaseOrders planPurchaseOrderCreator
	LineItems      planLineItemCreator
	Outbox         eventrecord.Enqueuer
}

// CreatePurchaseOrderFromPlanUseCase turns a supplier's share of a plan
// into one draft purchase order, atomically. Price and pack size are
// taken from the newest PO line for the SKU with this same supplier --
// never from the client -- and left 0 / default when it was last bought
// elsewhere (another supplier's price and box size don't apply). The
// currency is this supplier's last PO currency, else the tenant default.
type CreatePurchaseOrderFromPlanUseCase struct {
	db   *gorm.DB
	deps PlanOrderDeps
}

func NewCreatePurchaseOrderFromPlanUseCase(db *gorm.DB, deps PlanOrderDeps) *CreatePurchaseOrderFromPlanUseCase {
	return &CreatePurchaseOrderFromPlanUseCase{db: db, deps: deps}
}

func (uc *CreatePurchaseOrderFromPlanUseCase) Execute(ctx context.Context, in PlanOrderInput) (*procurementDomain.PurchaseOrder, error) {
	if err := validatePlanOrder(in); err != nil {
		return nil, fmt.Errorf("createPurchaseOrderFromPlan: %w", err)
	}
	var po procurementDomain.PurchaseOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.deps.Suppliers.FindByID(ctx, in.SupplierID); err != nil {
			return err
		}
		plannable, err := uc.plannableSKUs(ctx)
		if err != nil {
			return err
		}
		for _, line := range in.Lines {
			if _, ok := plannable[line.SKUID]; !ok {
				return fmt.Errorf("%s: %w", line.SKUID, ErrSKUNotPlannable)
			}
		}
		sourcing, err := uc.deps.Supply.LatestSourcing(ctx)
		if err != nil {
			return err
		}
		currency, err := uc.currency(ctx, in, sourcing)
		if err != nil {
			return err
		}

		now := time.Now()
		supplierID := in.SupplierID
		po = procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: &supplierID,
			Status: procurementDomain.PurchaseOrderStatusDraft, CurrencyCode: currency,
			Notes: in.Notes, CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.deps.PurchaseOrders.Create(ctx, &po); err != nil {
			return err
		}
		for i, line := range in.Lines {
			sku := plannable[line.SKUID]
			perItem, cost := planLinePack(sku, sourcing[line.SKUID], in.SupplierID)
			skuID := line.SKUID
			li := procurementDomain.PurchaseOrderLineItem{
				TenantID: po.TenantID, PurchaseOrderID: po.ID, SKUID: &skuID, RawDescription: sku.ProductName,
				Position: int64(i), Quantity: line.Quantity, UnitsPerPurchasedItem: perItem, UnitCostMinorUnits: cost,
				LineType: procurementDomain.LineItemTypeProduct, LineTypeConfirmed: true,
				CreatedAt: now, UpdatedAt: now,
			}
			if err := uc.deps.LineItems.Create(ctx, &li); err != nil {
				return err
			}
		}
		return uc.enqueueCreated(ctx, po, len(in.Lines))
	})
	if err != nil {
		return nil, fmt.Errorf("createPurchaseOrderFromPlan: %w", err)
	}
	return &po, nil
}

// enqueueCreated records procurement.purchase_order.created, atomic with
// the PO (same tenant.WithTenant tx).
func (uc *CreatePurchaseOrderFromPlanUseCase) enqueueCreated(ctx context.Context, po procurementDomain.PurchaseOrder, lineCount int) error {
	return eventrecord.Record(ctx, uc.deps.Outbox, events.PurchaseOrderCreated, po.ID, eventrecord.Data{
		"source": PurchaseOrderSourcePlanner, "supplier_id": *po.SupplierID,
		"currency_code": po.CurrencyCode, "line_count": lineCount,
	})
}

func validatePlanOrder(in PlanOrderInput) error {
	if len(in.Lines) == 0 {
		return ErrInvalidPlanOrder
	}
	seen := make(map[string]bool, len(in.Lines))
	for _, l := range in.Lines {
		if l.Quantity <= 0 || seen[l.SKUID] {
			return ErrInvalidPlanOrder
		}
		seen[l.SKUID] = true
	}
	return nil
}

func (uc *CreatePurchaseOrderFromPlanUseCase) plannableSKUs(ctx context.Context) (map[string]planning.PlanningSKU, error) {
	skus, err := uc.deps.Catalog.ListPlanningSKUs(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]planning.PlanningSKU{}
	for _, s := range planning.NewRollup(skus).Plannable() {
		out[s.SKUID] = s
	}
	return out, nil
}

func (uc *CreatePurchaseOrderFromPlanUseCase) currency(ctx context.Context, in PlanOrderInput, sourcing map[string]planning.Sourcing) (string, error) {
	for _, l := range in.Lines {
		if s, ok := sourcing[l.SKUID]; ok && s.SupplierID == in.SupplierID && s.CurrencyCode != "" {
			return s.CurrencyCode, nil
		}
	}
	settings, err := uc.deps.Settings.Get(ctx)
	if err != nil {
		return "", err
	}
	return settings.DefaultCurrency, nil
}

// planLinePack is a line's units per purchased item and unit cost: from
// the SKU's last PO line when that was with this supplier, else defaults.
func planLinePack(sku planning.PlanningSKU, last planning.Sourcing, supplierID string) (int64, int64) {
	if last.SupplierID == supplierID && last.UnitsPerPurchasedItem > 0 {
		return last.UnitsPerPurchasedItem, last.UnitCostMinorUnits
	}
	if sku.IsWeightPool {
		return weightPoolUnitsPerPurchasedItem, 0
	}
	return 1, 0
}
