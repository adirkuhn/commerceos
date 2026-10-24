package oms

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// LineChangeSources are the fulfilment steps that apply order changes.
type LineChangeSources struct {
	Packing omsDomain.AppliedChangeLister
	Picking omsDomain.AppliedChangeLister
}

type lineChangeLineLister interface {
	ListBySalesOrder(ctx context.Context, salesOrderID string) ([]omsDomain.SalesOrderLineItem, error)
}

// SalesOrderLineChange is one applied order change, described for the order
// page: a line replaced by another product, an extra item added (free or
// charged) or part of a line refunded.
type SalesOrderLineChange struct {
	ID     string
	Step   OrderChangeStep
	Action omsDomain.ChangeAction
	// LineItemID is the line replaced or refunded, or the line an extra
	// item became; nil for an extra that was never added (no stock).
	LineItemID *string
	// SKUID/SKUCode/ProductName/VariantName are the product packed: the
	// substitute, the extra item, or the refunded line's own SKU. Empty
	// when the SKU no longer exists.
	SKUID              string
	SKUCode            string
	ProductName        string
	VariantName        string
	Quantity           int64
	OriginalQtyCovered *int64
	ReturnToStock      bool
	// PriceMinorUnits is a charged extra item's unit price.
	PriceMinorUnits *int64
	// Reason is staff's note; for an extra item, the name it was added as.
	Reason string
	// By is the lm2 UserID who recorded the change.
	By string
	At time.Time
}

// ListSalesOrderLineChangesUseCase lists every order change picking and
// packing applied to a sales order, oldest first.
type ListSalesOrderLineChangesUseCase struct {
	db          *gorm.DB
	sources     LineChangeSources
	lines       lineChangeLineLister
	skuRepo     skuFinderForPickingImages
	variantRepo variantFinderForPickScanResolve
	productRepo productFinderForPickScanResolve
}

func NewListSalesOrderLineChangesUseCase(
	db *gorm.DB,
	sources LineChangeSources,
	lines lineChangeLineLister,
	skuRepo skuFinderForPickingImages,
	variantRepo variantFinderForPickScanResolve,
	productRepo productFinderForPickScanResolve,
) *ListSalesOrderLineChangesUseCase {
	return &ListSalesOrderLineChangesUseCase{db: db, sources: sources, lines: lines, skuRepo: skuRepo, variantRepo: variantRepo, productRepo: productRepo}
}

func (uc *ListSalesOrderLineChangesUseCase) Execute(ctx context.Context, salesOrderID string) ([]SalesOrderLineChange, error) {
	var out []SalesOrderLineChange
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		picked, err := uc.sources.Picking.ListAppliedBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}
		packed, err := uc.sources.Packing.ListAppliedBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}
		lines, err := uc.lines.ListBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}

		changes := append(withStep(picked, OrderChangeStepPicking), withStep(packed, OrderChangeStepPacking)...)
		sort.SliceStable(changes, func(i, j int) bool { return changes[i].ScannedAt.Before(changes[j].ScannedAt) })
		all := make([]omsDomain.StagedChange, len(changes))
		for i, c := range changes {
			all[i] = c.StagedChange
		}
		addedLines := omsDomain.MatchAddedLines(all, lines)

		products := make(map[string]skuProduct)
		for _, c := range changes {
			p, ok := products[c.SKUID]
			if !ok {
				if p, err = uc.describeSKU(ctx, c.SKUID); err != nil {
					return err
				}
				products[c.SKUID] = p
			}
			out = append(out, toLineChange(c, p, addedLines))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listSalesOrderLineChanges: %w", err)
	}
	return out, nil
}

type stepChange struct {
	omsDomain.StagedChange
	step OrderChangeStep
}

func withStep(changes []omsDomain.StagedChange, step OrderChangeStep) []stepChange {
	out := make([]stepChange, len(changes))
	for i, c := range changes {
		out[i] = stepChange{StagedChange: c, step: step}
	}
	return out
}

type skuProduct struct{ code, product, variant string }

// describeSKU names a SKU by its product and variant; a SKU deleted since
// stays nameless rather than hiding the change.
func (uc *ListSalesOrderLineChangesUseCase) describeSKU(ctx context.Context, skuID string) (skuProduct, error) {
	sku, err := uc.skuRepo.FindByID(ctx, skuID)
	if errors.Is(err, pmsDomain.ErrSKUNotFound) || (err == nil && sku == nil) {
		return skuProduct{}, nil
	}
	if err != nil {
		return skuProduct{}, err
	}
	p := skuProduct{code: sku.SKUCode}
	variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID)
	if err != nil {
		return skuProduct{}, err
	}
	product, err := uc.productRepo.FindByID(ctx, variant.ProductID)
	if err != nil {
		return skuProduct{}, err
	}
	p.variant, p.product = variant.Name, product.Name
	return p, nil
}

func toLineChange(c stepChange, p skuProduct, addedLines map[string]string) SalesOrderLineChange {
	lineItemID := c.LineItemID
	if c.IsExtraItem() {
		lineItemID = nil
		if id, ok := addedLines[c.ID]; ok {
			lineItemID = &id
		}
	}
	return SalesOrderLineChange{
		ID: c.ID, Step: c.step, Action: c.Action, LineItemID: lineItemID,
		SKUID: c.SKUID, SKUCode: p.code, ProductName: p.product, VariantName: p.variant,
		Quantity: c.Quantity, OriginalQtyCovered: c.OriginalQtyCovered, ReturnToStock: c.ReturnToStock,
		PriceMinorUnits: c.PriceMinorUnits, Reason: c.Reason, By: c.ScannedBy, At: c.ScannedAt,
	}
}
