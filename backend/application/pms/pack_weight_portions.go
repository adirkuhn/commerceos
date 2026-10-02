package pms

import (
	"context"
	"fmt"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// WeightPackageLine is one row of "N packages of W grams" weighed off a
// weight-pool bulk SKU.
type WeightPackageLine struct {
	WeightGrams int64
	Quantity    int64
}

// PackWeightPortionsInput describes packages just weighed off a bulk
// weight-pool SKU (see pms.SKU.UnitType=kilo), e.g. "1 x 430g, 4 x 540g,
// 10 x 380g" -- the "split this lot into sellable packages" flow.
type PackWeightPortionsInput struct {
	BulkSKUID   string
	WarehouseID string
	// BulkBatchID, nil for general/un-lotted bulk stock, is the lot these
	// packages come from -- its LotNumber/ExpiryDate are carried onto a
	// batch of each package SKU (see application/channels' toWooMetaData,
	// which pushes a SKU's batch expiry to WooCommerce).
	BulkBatchID *string
	Lines       []WeightPackageLine
	// SKUCodePrefix defaults to the bulk SKU's own SKUCode when blank --
	// a newly created package SKU's code is "{prefix}-{weight}G".
	SKUCodePrefix string
	Reason        string
}

// PackedWeightLine reports what happened to one distinct weight: the
// package SKU it landed on, whether that SKU was created by this call or
// already existed (restocked), and how many packages were added.
type PackedWeightLine struct {
	SKU         pmsDomain.SKU
	WeightGrams int64
	Quantity    int64
	Created     bool
}

type PackWeightPortionsResult struct {
	Lines []PackedWeightLine
}

// batchReaderForPack/batchCreatorForPack/portionerForPack/skuCreatorForPack/
// scopedStockReaderForPack are PackWeightPortionsUseCase's collaborators,
// narrowed to point of use (ISP).
type batchReaderForPack interface {
	FindByID(ctx context.Context, id string) (*invDomain.Batch, error)
	ListBySKU(ctx context.Context, skuID string) ([]invDomain.Batch, error)
}
type batchCreatorForPack interface {
	Execute(ctx context.Context, in invapp.CreateBatchInput) (*invDomain.Batch, error)
}
type portionerForPack interface {
	Execute(ctx context.Context, in invapp.PortionFromBulkInput) (*invapp.PortionFromBulkResult, error)
}
type skuCreatorForPack interface {
	Execute(ctx context.Context, in CreateSKUInput) (*pmsDomain.SKU, error)
}
type scopedStockReaderForPack interface {
	Execute(ctx context.Context, skuID, warehouseID string, batchID *string) (int64, error)
}

// PackWeightPortionsUseCase turns weighed packages into sellable stock:
// one package SKU per distinct weight (UnitType=unit, PricingMode=fixed,
// PortionedFromSKUID = the bulk pool, priced from the bulk's
// PricePerKiloMinorUnits, barcoded via pms.EncodeWeightEmbeddedBarcode --
// the exact label the scale prints for that weight), stocked with that
// line's quantity. A weight that already has a package SKU off this pool
// is restocked rather than re-created, so the same weight can be packed
// again in a later session.
//
// The whole list is checked against the chosen lot's remaining stock up
// front, so a list that doesn't fit changes nothing. After that, like
// CompletePickingUseCase's replay, it composes use cases that each open
// their own transaction (tenant.WithTenant must never nest) -- one
// find-or-create + portion cycle per weight; an unexpected failure partway
// names the weight that failed, earlier weights stay packed.
type PackWeightPortionsUseCase struct {
	db          *gorm.DB
	skuRepo     pmsDomain.SKURepository
	batchRepo   batchReaderForPack
	createSKU   skuCreatorForPack
	createBatch batchCreatorForPack
	portion     portionerForPack
	stock       scopedStockReaderForPack
}

func NewPackWeightPortionsUseCase(
	db *gorm.DB, skuRepo pmsDomain.SKURepository, batchRepo batchReaderForPack,
	createSKU skuCreatorForPack, createBatch batchCreatorForPack, portion portionerForPack, stock scopedStockReaderForPack,
) *PackWeightPortionsUseCase {
	return &PackWeightPortionsUseCase{
		db: db, skuRepo: skuRepo, batchRepo: batchRepo,
		createSKU: createSKU, createBatch: createBatch, portion: portion, stock: stock,
	}
}

func (uc *PackWeightPortionsUseCase) Execute(ctx context.Context, in PackWeightPortionsInput) (*PackWeightPortionsResult, error) {
	lines, err := mergeWeightLines(in.Lines)
	if err != nil {
		return nil, fmt.Errorf("packWeightPortions: %w", err)
	}

	bulkSKU, bulkBatch, existing, err := uc.loadBulk(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("packWeightPortions: %w", err)
	}
	if err := uc.requireLotCovers(ctx, in, lines); err != nil {
		return nil, fmt.Errorf("packWeightPortions: %w", err)
	}

	codePrefix := in.SKUCodePrefix
	if codePrefix == "" {
		codePrefix = bulkSKU.SKUCode
	}

	result := &PackWeightPortionsResult{Lines: make([]PackedWeightLine, 0, len(lines))}
	for _, line := range lines {
		packed, err := uc.packLine(ctx, in, *bulkSKU, bulkBatch, existing[line.WeightGrams], codePrefix, line)
		if err != nil {
			return nil, fmt.Errorf("packWeightPortions: packages at %dg: %w", line.WeightGrams, err)
		}
		result.Lines = append(result.Lines, *packed)
	}
	return result, nil
}

// mergeWeightLines validates the lines and folds repeated weights into one
// line each, keeping first-appearance order ("3 x 380g" then "2 x 380g" is
// one 380g package SKU with 5 added).
func mergeWeightLines(in []WeightPackageLine) ([]WeightPackageLine, error) {
	if len(in) == 0 {
		return nil, pmsDomain.ErrNoWeightsProvided
	}
	merged := make([]WeightPackageLine, 0, len(in))
	index := make(map[int64]int, len(in))
	for _, l := range in {
		if l.WeightGrams <= 0 || l.Quantity <= 0 {
			return nil, invDomain.ErrInvalidQuantity
		}
		if i, ok := index[l.WeightGrams]; ok {
			merged[i].Quantity += l.Quantity
			continue
		}
		index[l.WeightGrams] = len(merged)
		merged = append(merged, l)
	}
	return merged, nil
}

// loadBulk reads the bulk SKU, the chosen lot, and the package SKUs
// already portioned off this pool, keyed by weight.
func (uc *PackWeightPortionsUseCase) loadBulk(ctx context.Context, in PackWeightPortionsInput) (*pmsDomain.SKU, *invDomain.Batch, map[int64]*pmsDomain.SKU, error) {
	var bulkSKU *pmsDomain.SKU
	var bulkBatch *invDomain.Batch
	existing := map[int64]*pmsDomain.SKU{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByID(ctx, in.BulkSKUID)
		if err != nil {
			return err
		}
		if !isScaleSoldWeightPool(*sku) {
			return pmsDomain.ErrBulkSKUNotWeightPool
		}
		bulkSKU = sku
		if in.BulkBatchID != nil {
			b, err := uc.batchRepo.FindByID(ctx, *in.BulkBatchID)
			if err != nil {
				return err
			}
			if b.SKUID != in.BulkSKUID {
				return invDomain.ErrBatchDoesNotBelongToSKU
			}
			bulkBatch = b
		}
		siblings, err := uc.skuRepo.ListByVariant(ctx, sku.VariantID)
		if err != nil {
			return err
		}
		for i := range siblings {
			s := &siblings[i]
			if s.PortionedFromSKUID == nil || *s.PortionedFromSKUID != sku.ID || s.WeightGrams == nil {
				continue
			}
			// Prefer an active package SKU over e.g. an archived one.
			if prev, ok := existing[*s.WeightGrams]; ok && prev.Status == pmsDomain.SKUStatusActive {
				continue
			}
			existing[*s.WeightGrams] = s
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return bulkSKU, bulkBatch, existing, nil
}

func isScaleSoldWeightPool(s pmsDomain.SKU) bool {
	return s.UnitType == pmsDomain.SKUUnitTypeKilo && s.PricingMode == pmsDomain.SKUPricingModePerWeight &&
		s.PricePerKiloMinorUnits != nil && s.ScaleCode != nil && *s.ScaleCode != ""
}

func (uc *PackWeightPortionsUseCase) requireLotCovers(ctx context.Context, in PackWeightPortionsInput, lines []WeightPackageLine) error {
	var needed int64
	for _, l := range lines {
		needed += l.WeightGrams * l.Quantity
	}
	available, err := uc.stock.Execute(ctx, in.BulkSKUID, in.WarehouseID, in.BulkBatchID)
	if err != nil {
		return err
	}
	if needed > available {
		return invDomain.ErrInsufficientStock
	}
	return nil
}

func (uc *PackWeightPortionsUseCase) packLine(
	ctx context.Context, in PackWeightPortionsInput, bulkSKU pmsDomain.SKU, bulkBatch *invDomain.Batch,
	existing *pmsDomain.SKU, codePrefix string, line WeightPackageLine,
) (*PackedWeightLine, error) {
	packageSKU, created := existing, false
	if packageSKU == nil {
		sku, err := uc.createPackageSKU(ctx, bulkSKU, codePrefix, line.WeightGrams)
		if err != nil {
			return nil, err
		}
		packageSKU, created = sku, true
	}

	var targetBatchID *string
	if bulkBatch != nil {
		id, err := uc.packageBatchFor(ctx, in.WarehouseID, packageSKU.ID, !created, *bulkBatch)
		if err != nil {
			return nil, err
		}
		targetBatchID = &id
	}

	w := line.WeightGrams
	if _, err := uc.portion.Execute(ctx, invapp.PortionFromBulkInput{
		TargetSKUID: packageSKU.ID, WarehouseID: in.WarehouseID, BatchID: in.BulkBatchID, WeightGrams: &w,
		Quantity: line.Quantity, TargetBatchID: targetBatchID, Reason: in.Reason, OccurredAt: time.Now(),
	}); err != nil {
		return nil, err
	}
	return &PackedWeightLine{SKU: *packageSKU, WeightGrams: line.WeightGrams, Quantity: line.Quantity, Created: created}, nil
}

func (uc *PackWeightPortionsUseCase) createPackageSKU(ctx context.Context, bulkSKU pmsDomain.SKU, codePrefix string, weight int64) (*pmsDomain.SKU, error) {
	barcode, err := pmsDomain.EncodeWeightEmbeddedBarcode(*bulkSKU.ScaleCode, weight)
	if err != nil {
		return nil, err
	}
	w := weight
	return uc.createSKU.Execute(ctx, CreateSKUInput{
		VariantID: bulkSKU.VariantID, SKUCode: fmt.Sprintf("%s-%dG", codePrefix, weight), Barcode: barcode,
		PriceMinorUnits: lineTotalForScannedWeight(bulkSKU, weight), CurrencyCode: bulkSKU.CurrencyCode, WeightGrams: &w,
		UnitType: pmsDomain.SKUUnitTypeUnit, PortionedFromSKUID: &bulkSKU.ID,
		PricingMode: pmsDomain.SKUPricingModeFixed, Status: pmsDomain.SKUStatusActive,
	})
}

// packageBatchFor returns the package SKU's batch mirroring the bulk lot
// (same warehouse, lot number and expiry), creating it when the SKU has
// none yet -- packing the same lot again reuses it instead of piling up
// identical batches.
func (uc *PackWeightPortionsUseCase) packageBatchFor(ctx context.Context, warehouseID, packageSKUID string, mayHaveBatches bool, bulkBatch invDomain.Batch) (string, error) {
	if mayHaveBatches {
		var found string
		err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
			batches, err := uc.batchRepo.ListBySKU(ctx, packageSKUID)
			if err != nil {
				return err
			}
			for _, b := range batches {
				if b.WarehouseID == warehouseID && b.Status == invDomain.BatchStatusActive &&
					b.LotNumber == bulkBatch.LotNumber && sameExpiry(b.ExpiryDate, bulkBatch.ExpiryDate) {
					found = b.ID
					return nil
				}
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		if found != "" {
			return found, nil
		}
	}
	created, err := uc.createBatch.Execute(ctx, invapp.CreateBatchInput{
		SKUID: packageSKUID, WarehouseID: warehouseID, LotNumber: bulkBatch.LotNumber, ExpiryDate: bulkBatch.ExpiryDate,
	})
	if err != nil {
		return "", err
	}
	return created.ID, nil
}

func sameExpiry(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}
