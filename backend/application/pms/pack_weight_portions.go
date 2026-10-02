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

// The ports below are PackWeightPortionsUseCase's collaborators, narrowed
// to point of use (ISP) -- they let it compose this package's
// CreateVariant/CreateSKU/UpdateProduct and application/inventory's
// CreateBatch/PortionFromBulk/GetScopedStock without their concrete types.
type skuReaderForPack interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error)
	ListByVariant(ctx context.Context, variantID string) ([]pmsDomain.SKU, error)
}
type variantReaderForPack interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Variant, error)
	ListByProduct(ctx context.Context, productID string) ([]pmsDomain.Variant, error)
}
type productReaderForPack interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Product, error)
}
type batchReaderForPack interface {
	FindByID(ctx context.Context, id string) (*invDomain.Batch, error)
	ListBySKU(ctx context.Context, skuID string) ([]invDomain.Batch, error)
}
type variantCreatorForPack interface {
	Execute(ctx context.Context, in CreateVariantInput) (*pmsDomain.Variant, error)
}
type productUpdaterForPack interface {
	Execute(ctx context.Context, in UpdateProductInput) (*pmsDomain.Product, error)
}
type skuCreatorForPack interface {
	Execute(ctx context.Context, in CreateSKUInput) (*pmsDomain.SKU, error)
}
type batchCreatorForPack interface {
	Execute(ctx context.Context, in invapp.CreateBatchInput) (*invDomain.Batch, error)
}
type portionerForPack interface {
	Execute(ctx context.Context, in invapp.PortionFromBulkInput) (*invapp.PortionFromBulkResult, error)
}
type scopedStockReaderForPack interface {
	Execute(ctx context.Context, skuID, warehouseID string, batchID *string) (int64, error)
}

type PackWeightPortionsDeps struct {
	SKUs           skuReaderForPack
	Variants       variantReaderForPack
	Products       productReaderForPack
	Batches        batchReaderForPack
	CreateVariant  variantCreatorForPack
	EnableVariants productUpdaterForPack
	CreateSKU      skuCreatorForPack
	CreateBatch    batchCreatorForPack
	Portion        portionerForPack
	Stock          scopedStockReaderForPack
}

// PackWeightPortionsUseCase turns weighed packages into sellable stock.
// Each distinct weight gets its own variant ("~380g") on the bulk SKU's
// product holding exactly one package SKU (UnitType=unit,
// PricingMode=fixed, PortionedFromSKUID = the bulk pool, priced by
// pmsDomain.PackagePriceMinorUnits from the bulk's rate, barcoded via
// pms.EncodeWeightEmbeddedBarcode -- the exact label the scale prints for
// that weight), stocked with that line's quantity. One SKU per variant is
// what every sales-channel push requires (see application/channels'
// PushProductToWooUseCase), so a simple product is switched to variants
// mode before its first package variant is added. A weight that already
// has a package off this pool is restocked rather than re-created, so the
// same weight can be packed again in a later session.
//
// The whole list is checked against the chosen lot's remaining stock up
// front, so a list that doesn't fit changes nothing. After that, like
// CompletePickingUseCase's replay, it composes use cases that each open
// their own transaction (tenant.WithTenant must never nest) -- one
// find-or-create + portion cycle per weight; an unexpected failure partway
// names the weight that failed, earlier weights stay packed.
type PackWeightPortionsUseCase struct {
	db   *gorm.DB
	deps PackWeightPortionsDeps
}

func NewPackWeightPortionsUseCase(db *gorm.DB, deps PackWeightPortionsDeps) *PackWeightPortionsUseCase {
	return &PackWeightPortionsUseCase{db: db, deps: deps}
}

// packSource is everything Execute reads up front: the bulk SKU and its
// product, the chosen lot, the product's variant count (new variants are
// positioned after it), and the packages already portioned off this pool,
// keyed by weight.
type packSource struct {
	bulkSKU      pmsDomain.SKU
	product      pmsDomain.Product
	bulkBatch    *invDomain.Batch
	variantCount int
	existing     map[int64]*pmsDomain.SKU
}

func (uc *PackWeightPortionsUseCase) Execute(ctx context.Context, in PackWeightPortionsInput) (*PackWeightPortionsResult, error) {
	lines, err := mergeWeightLines(in.Lines)
	if err != nil {
		return nil, fmt.Errorf("packWeightPortions: %w", err)
	}
	src, err := uc.loadSource(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("packWeightPortions: %w", err)
	}
	if err := uc.requireLotCovers(ctx, in, lines); err != nil {
		return nil, fmt.Errorf("packWeightPortions: %w", err)
	}
	if err := uc.enableVariantsIfNeeded(ctx, src, lines); err != nil {
		return nil, fmt.Errorf("packWeightPortions: %w", err)
	}

	codePrefix := in.SKUCodePrefix
	if codePrefix == "" {
		codePrefix = src.bulkSKU.SKUCode
	}

	result := &PackWeightPortionsResult{Lines: make([]PackedWeightLine, 0, len(lines))}
	for _, line := range lines {
		packed, err := uc.packLine(ctx, in, src, codePrefix, line)
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

func (uc *PackWeightPortionsUseCase) loadSource(ctx context.Context, in PackWeightPortionsInput) (*packSource, error) {
	src := &packSource{existing: map[int64]*pmsDomain.SKU{}}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.deps.SKUs.FindByID(ctx, in.BulkSKUID)
		if err != nil {
			return err
		}
		if !isScaleSoldWeightPool(*sku) {
			return pmsDomain.ErrBulkSKUNotWeightPool
		}
		src.bulkSKU = *sku
		if in.BulkBatchID != nil {
			b, err := uc.deps.Batches.FindByID(ctx, *in.BulkBatchID)
			if err != nil {
				return err
			}
			if b.SKUID != in.BulkSKUID {
				return invDomain.ErrBatchDoesNotBelongToSKU
			}
			src.bulkBatch = b
		}
		variant, err := uc.deps.Variants.FindByID(ctx, sku.VariantID)
		if err != nil {
			return err
		}
		product, err := uc.deps.Products.FindByID(ctx, variant.ProductID)
		if err != nil {
			return err
		}
		src.product = *product
		variants, err := uc.deps.Variants.ListByProduct(ctx, product.ID)
		if err != nil {
			return err
		}
		src.variantCount = len(variants)
		for _, v := range variants {
			// A package SKU sharing the pool's own variant (how the
			// previous bulk tool made them) can never be pushed to a
			// sales channel -- not a package to restock.
			if v.ID == sku.VariantID {
				continue
			}
			skus, err := uc.deps.SKUs.ListByVariant(ctx, v.ID)
			if err != nil {
				return err
			}
			if len(skus) != 1 {
				continue
			}
			s := skus[0]
			if s.PortionedFromSKUID == nil || *s.PortionedFromSKUID != sku.ID || s.WeightGrams == nil {
				continue
			}
			// Prefer an active package SKU over e.g. an archived one.
			if prev, ok := src.existing[*s.WeightGrams]; ok && prev.Status == pmsDomain.SKUStatusActive {
				continue
			}
			src.existing[*s.WeightGrams] = &s
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return src, nil
}

// enableVariantsIfNeeded switches a simple product to variants mode before
// its first package variant is created -- only HasVariants changes, every
// other field is passed through as-is.
func (uc *PackWeightPortionsUseCase) enableVariantsIfNeeded(ctx context.Context, src *packSource, lines []WeightPackageLine) error {
	if src.product.HasVariants {
		return nil
	}
	needsNewVariant := false
	for _, l := range lines {
		if src.existing[l.WeightGrams] == nil {
			needsNewVariant = true
			break
		}
	}
	if !needsNewVariant {
		return nil
	}
	p := src.product
	_, err := uc.deps.EnableVariants.Execute(ctx, UpdateProductInput{
		ID: p.ID, Name: p.Name, Description: p.Description, CategoryID: p.CategoryID, Status: p.Status,
		HasVariants: true, SyncVariantsAsWooVariations: p.SyncVariantsAsWooVariations, Tags: p.Tags,
	})
	return err
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
	available, err := uc.deps.Stock.Execute(ctx, in.BulkSKUID, in.WarehouseID, in.BulkBatchID)
	if err != nil {
		return err
	}
	if needed > available {
		return invDomain.ErrInsufficientStock
	}
	return nil
}

func (uc *PackWeightPortionsUseCase) packLine(
	ctx context.Context, in PackWeightPortionsInput, src *packSource, codePrefix string, line WeightPackageLine,
) (*PackedWeightLine, error) {
	packageSKU, created := src.existing[line.WeightGrams], false
	if packageSKU == nil {
		sku, err := uc.createPackage(ctx, src, codePrefix, line.WeightGrams)
		if err != nil {
			return nil, err
		}
		packageSKU, created = sku, true
	}

	var targetBatchID *string
	if src.bulkBatch != nil {
		id, err := uc.packageBatchFor(ctx, in.WarehouseID, packageSKU.ID, !created, *src.bulkBatch)
		if err != nil {
			return nil, err
		}
		targetBatchID = &id
	}

	w := line.WeightGrams
	if _, err := uc.deps.Portion.Execute(ctx, invapp.PortionFromBulkInput{
		TargetSKUID: packageSKU.ID, WarehouseID: in.WarehouseID, BatchID: in.BulkBatchID, WeightGrams: &w,
		Quantity: line.Quantity, TargetBatchID: targetBatchID, Reason: in.Reason, OccurredAt: time.Now(),
	}); err != nil {
		return nil, err
	}
	return &PackedWeightLine{SKU: *packageSKU, WeightGrams: line.WeightGrams, Quantity: line.Quantity, Created: created}, nil
}

// createPackage adds a "~{weight}g" variant after the product's existing
// ones, holding the package SKU for that weight.
func (uc *PackWeightPortionsUseCase) createPackage(ctx context.Context, src *packSource, codePrefix string, weight int64) (*pmsDomain.SKU, error) {
	barcode, err := pmsDomain.EncodeWeightEmbeddedBarcode(*src.bulkSKU.ScaleCode, weight)
	if err != nil {
		return nil, err
	}
	variant, err := uc.deps.CreateVariant.Execute(ctx, CreateVariantInput{
		ProductID: src.product.ID, Name: fmt.Sprintf("~%dg", weight), Position: src.variantCount,
	})
	if err != nil {
		return nil, err
	}
	src.variantCount++
	w := weight
	bulk := src.bulkSKU
	return uc.deps.CreateSKU.Execute(ctx, CreateSKUInput{
		VariantID: variant.ID, SKUCode: fmt.Sprintf("%s-%dG", codePrefix, weight), Barcode: barcode,
		PriceMinorUnits: pmsDomain.PackagePriceMinorUnits(*bulk.PricePerKiloMinorUnits, weight), CurrencyCode: bulk.CurrencyCode, WeightGrams: &w,
		UnitType: pmsDomain.SKUUnitTypeUnit, PortionedFromSKUID: &bulk.ID,
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
			batches, err := uc.deps.Batches.ListBySKU(ctx, packageSKUID)
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
	created, err := uc.deps.CreateBatch.Execute(ctx, invapp.CreateBatchInput{
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
