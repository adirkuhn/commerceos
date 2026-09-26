package pms

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// SKUReferenceReassigner moves every reference one module holds to a SKU
// (stock, batches, order lines, receiving scans, aliases, ...) onto another
// SKU. Each module implements it for its own tables. It runs on the
// caller's transaction (from ctx), never its own: a merge must move all of
// it or none of it -- which is why a merge is one transaction and not a
// chain of events (a half-applied merge would leave stock split across a
// deleted and a live SKU).
type SKUReferenceReassigner interface {
	ReassignSKU(ctx context.Context, fromSKUID, toSKUID string) error
}

// ProductReferenceReassigner is SKUReferenceReassigner for product-level
// references (images, reviews, ...).
type ProductReferenceReassigner interface {
	ReassignProduct(ctx context.Context, fromProductID, toProductID string) error
}

type mergeProductStore interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Product, error)
	Delete(ctx context.Context, id string) error
}

type mergeVariantLister interface {
	ListByProduct(ctx context.Context, productID string) ([]pmsDomain.Variant, error)
}

type mergeSKULister interface {
	ListByVariant(ctx context.Context, variantID string) ([]pmsDomain.SKU, error)
}

type mergeSKUStore interface {
	mergeSKULister
	UpdateCostPriceFromReceipt(ctx context.Context, skuID string, costPriceMinorUnits int64) error
}

// mergeOnHand is inventory's ListBatchesUseCase narrowed to point of use:
// a SKU's on-hand quantity is the sum of its batch rows (general ledger
// included) -- read through inventory's own use case, not its repository.
type mergeOnHand interface {
	Execute(ctx context.Context, skuID string) ([]invDomain.Batch, error)
}

type mergeLedger interface {
	Create(ctx context.Context, e *pmsDomain.SKUCostPriceLedgerEntry) error
}

// mergeCodeAliases keeps a duplicate's barcode/scale code scanning to the
// existing SKU once the duplicate is gone.
type mergeCodeAliases interface {
	// CodeTaken reports whether any SKU or alias already answers to code.
	CodeTaken(ctx context.Context, code string) (bool, error)
	CreateBarcodeAlias(ctx context.Context, a *pmsDomain.BarcodeAlias) error
	CreateScaleCodeAlias(ctx context.Context, a *pmsDomain.ScaleCodeAlias) error
}

type mergeRecordStore interface {
	Create(ctx context.Context, r *pmsDomain.ProductMergeRecord) error
}

type mergeListingLister interface {
	ListByProduct(ctx context.Context, productID string) ([]channelsDomain.ProductChannelListing, error)
}

// MergeProductsDeps groups MergeProductsUseCase's ports -- more than a
// constructor's positional parameters can keep readable.
type MergeProductsDeps struct {
	Products           mergeProductStore
	Variants           mergeVariantLister
	SKUs               mergeSKUStore
	OnHand             mergeOnHand
	SKUReassigners     []SKUReferenceReassigner
	ProductReassigners []ProductReferenceReassigner
	Ledger             mergeLedger
	CodeAliases        mergeCodeAliases
	Records            mergeRecordStore
	Listings           mergeListingLister
	Outbox             outbox.Repository
}

type MergeProductsInput struct {
	SourceProductID string
	TargetProductID string
	// Pairs may be empty when both products have exactly one SKU.
	Pairs    []pmsDomain.SKUMergePair
	MergedBy string
}

type MergeProductsResult struct {
	TargetProductID string
	SKUs            []pmsDomain.MergedSKU
}

// MergeProductsUseCase folds a duplicate product into an existing one (see
// pmsDomain.ProductMerge): per matched SKU pair, the duplicate's on-hand
// stock joins the existing SKU at its own cost (blended moving average,
// unless the existing SKU's cost is manual), every module moves its
// references, the duplicate's barcode/scale code become aliases of the
// existing SKU, and the duplicate product is deleted. The existing
// product's own data is never changed. One transaction; events after.
type MergeProductsUseCase struct {
	db   *gorm.DB
	deps MergeProductsDeps
}

func NewMergeProductsUseCase(db *gorm.DB, deps MergeProductsDeps) *MergeProductsUseCase {
	return &MergeProductsUseCase{db: db, deps: deps}
}

func (uc *MergeProductsUseCase) Execute(ctx context.Context, in MergeProductsInput) (*MergeProductsResult, error) {
	if !tenant.IsTenantAdmin(ctx) {
		return nil, fmt.Errorf("mergeProducts: %w", pmsDomain.ErrProductMergeForbidden)
	}
	if in.SourceProductID == in.TargetProductID {
		return nil, fmt.Errorf("mergeProducts: %w", pmsDomain.ErrMergeIntoSelf)
	}
	var result MergeProductsResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		source, err := uc.deps.Products.FindByID(ctx, in.SourceProductID)
		if err != nil {
			return err
		}
		if _, err := uc.deps.Products.FindByID(ctx, in.TargetProductID); err != nil {
			return err
		}
		sourceSKUs, err := listProductSKUs(ctx, uc.deps.Variants, uc.deps.SKUs, in.SourceProductID)
		if err != nil {
			return err
		}
		targetSKUs, err := listProductSKUs(ctx, uc.deps.Variants, uc.deps.SKUs, in.TargetProductID)
		if err != nil {
			return err
		}
		pairs := in.Pairs
		if len(pairs) == 0 {
			pairs, _ = pmsDomain.DefaultSKUMergePairs(sourceSKUs, targetSKUs)
		}
		merge, err := pmsDomain.NewProductMerge(in.SourceProductID, in.TargetProductID, sourceSKUs, targetSKUs, pairs)
		if err != nil {
			return err
		}

		bySKUID := make(map[string]pmsDomain.SKU, len(sourceSKUs)+len(targetSKUs))
		for _, s := range append(append([]pmsDomain.SKU{}, sourceSKUs...), targetSKUs...) {
			bySKUID[s.ID] = s
		}
		// On-hand is read before anything moves: afterwards the
		// duplicate's stock already counts toward the existing SKU.
		onHand := make(map[string]int64, len(bySKUID))
		for id := range bySKUID {
			if onHand[id], err = sumOnHand(ctx, uc.deps.OnHand, id); err != nil {
				return err
			}
		}

		for _, r := range uc.deps.ProductReassigners {
			if err := r.ReassignProduct(ctx, merge.SourceProductID, merge.TargetProductID); err != nil {
				return err
			}
		}
		for _, p := range merge.Pairs {
			for _, r := range uc.deps.SKUReassigners {
				if err := r.ReassignSKU(ctx, p.SourceSKUID, p.TargetSKUID); err != nil {
					return err
				}
			}
		}
		if err := uc.blendCosts(ctx, merge.Pairs, bySKUID, onHand); err != nil {
			return err
		}

		listings, err := uc.deps.Listings.ListByProduct(ctx, merge.SourceProductID)
		if err != nil {
			return err
		}
		if err := uc.deps.Products.Delete(ctx, merge.SourceProductID); err != nil {
			return err
		}
		// After the delete: the duplicate's own codes are free again.
		merged := make([]pmsDomain.MergedSKU, len(merge.Pairs))
		for i, p := range merge.Pairs {
			src := bySKUID[p.SourceSKUID]
			merged[i] = pmsDomain.MergedSKU{SourceSKUID: src.ID, SourceSKUCode: src.SKUCode, SourceBarcode: src.Barcode, TargetSKUID: p.TargetSKUID}
			if err := uc.keepCodes(ctx, src, bySKUID[p.TargetSKUID]); err != nil {
				return err
			}
		}

		if err := uc.deps.Records.Create(ctx, &pmsDomain.ProductMergeRecord{
			TenantID: tenant.IDFromContext(ctx), SourceProductID: source.ID, SourceProductName: source.Name,
			TargetProductID: merge.TargetProductID, SKUs: merged, MergedBy: in.MergedBy, MergedAt: time.Now(),
		}); err != nil {
			return err
		}
		result = MergeProductsResult{TargetProductID: merge.TargetProductID, SKUs: merged}
		return uc.enqueueEvents(ctx, merge, listings)
	})
	if err != nil {
		return nil, fmt.Errorf("mergeProducts: %w", err)
	}
	return &result, nil
}

// blendCosts writes each pair's folded cost (see foldMergePairs) and logs
// one merge ledger row per pair.
func (uc *MergeProductsUseCase) blendCosts(ctx context.Context, pairs []pmsDomain.SKUMergePair, bySKUID map[string]pmsDomain.SKU, onHand map[string]int64) error {
	for _, step := range foldMergePairs(pairs, bySKUID, onHand) {
		if step.costChanged {
			if err := uc.deps.SKUs.UpdateCostPriceFromReceipt(ctx, step.pair.TargetSKUID, *step.costAfter); err != nil {
				return err
			}
		}
		entry := &pmsDomain.SKUCostPriceLedgerEntry{
			TenantID: tenant.IDFromContext(ctx), SKUID: step.pair.TargetSKUID, SourceType: pmsDomain.SKUCostPriceLedgerSourceMerge,
			QuantityDelta: step.sourceQty, QuantityAfter: step.qtyAfter, CostPriceMinorUnits: step.costAfter,
			OccurredAt: time.Now(), CreatedAt: time.Now(),
		}
		if step.costAfter != nil {
			value := step.qtyAfter * *step.costAfter
			entry.InventoryValueMinorUnits = &value
		}
		if err := uc.deps.Ledger.Create(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

// keepCodes makes the duplicate's barcode and scale code alternates of the
// target SKU, unless something (the target itself included) already
// answers to them.
func (uc *MergeProductsUseCase) keepCodes(ctx context.Context, source, target pmsDomain.SKU) error {
	now := time.Now()
	if code := source.Barcode; code != "" && code != target.Barcode {
		taken, err := uc.deps.CodeAliases.CodeTaken(ctx, code)
		if err != nil {
			return err
		}
		if !taken {
			if err := uc.deps.CodeAliases.CreateBarcodeAlias(ctx, &pmsDomain.BarcodeAlias{
				TenantID: tenant.IDFromContext(ctx), SKUID: target.ID, Code: code, UnitsPerScan: 1, CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				return err
			}
		}
	}
	if source.ScaleCode != nil && *source.ScaleCode != "" && (target.ScaleCode == nil || *target.ScaleCode != *source.ScaleCode) {
		code := *source.ScaleCode
		taken, err := uc.deps.CodeAliases.CodeTaken(ctx, code)
		if err != nil {
			return err
		}
		if !taken {
			return uc.deps.CodeAliases.CreateScaleCodeAlias(ctx, &pmsDomain.ScaleCodeAlias{
				TenantID: tenant.IDFromContext(ctx), SKUID: target.ID, Code: code, CreatedAt: now, UpdatedAt: now,
			})
		}
	}
	return nil
}

func (uc *MergeProductsUseCase) enqueueEvents(ctx context.Context, merge *pmsDomain.ProductMerge, listings []channelsDomain.ProductChannelListing) error {
	skus := make([]events.ProductMergedSKU, len(merge.Pairs))
	targets := make(map[string]bool)
	for i, p := range merge.Pairs {
		skus[i] = events.ProductMergedSKU{SourceSKUID: p.SourceSKUID, TargetSKUID: p.TargetSKUID}
		targets[p.TargetSKUID] = true
	}
	merged := events.ProductMerged{
		Base:     events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()},
		TenantID: tenant.IDFromContext(ctx), SourceProductID: merge.SourceProductID, TargetProductID: merge.TargetProductID, SKUs: skus,
	}
	if err := enqueueEvent(ctx, uc.deps.Outbox, merged.EventName(), merged.EventID, merged, merged.OccurredAt); err != nil {
		return err
	}
	if err := enqueueProductDeleted(ctx, uc.deps.Outbox, merge.SourceProductID, listings); err != nil {
		return err
	}
	// The existing product gained images/reviews and stock: channels resync.
	if err := enqueueProductChanged(ctx, uc.deps.Outbox, merge.TargetProductID); err != nil {
		return err
	}
	for skuID := range targets {
		changed := events.InventoryStockChanged{
			Base:     events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()},
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID,
		}
		if err := enqueueEvent(ctx, uc.deps.Outbox, changed.EventName(), changed.EventID, changed, changed.OccurredAt); err != nil {
			return err
		}
	}
	return nil
}

func enqueueEvent(ctx context.Context, outboxRepo outbox.Repository, subject, eventID string, event any, occurredAt time.Time) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return outboxRepo.Enqueue(ctx, subject, eventID, payload, occurredAt)
}
