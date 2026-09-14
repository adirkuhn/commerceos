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

// BulkAddWeightPortionsInput describes a batch of physical packages just
// weighed off a bulk weight-pool SKU (see pms.SKU.UnitType=kilo), one entry
// in WeightsGrams per package -- the "type a list of weights, get N
// sellable items" flow, replacing what used to be one manual
// create-SKU-then-portion-from-bulk round trip per package.
type BulkAddWeightPortionsInput struct {
	BulkSKUID   string
	WarehouseID string
	// BulkBatchID, nil by default, is which lot of the bulk SKU these
	// packages are drawn from -- when set, its ExpiryDate/LotNumber are
	// copied onto a new Batch for each created package (see
	// application/channels' toWooMetaData, which already pushes a SKU's
	// batch lot/expiry to WooCommerce as product meta -- this is what
	// makes expiry show up on the sales channel for free). nil means
	// general/un-lotted bulk stock, and no expiry is carried forward.
	BulkBatchID  *string
	WeightsGrams []int64
	// SKUCodePrefix defaults to the bulk SKU's own SKUCode when blank --
	// each package's code becomes "{prefix}-{weight}G".
	SKUCodePrefix string
	Reason        string
}

type BulkAddWeightPortionsResult struct {
	Created []pmsDomain.SKU
}

// batchFinderForBulkAdd/batchCreatorForBulkAdd/portionerForBulkAdd/
// skuCreatorForBulkAdd are BulkAddWeightPortionsUseCase's collaborators,
// narrowed to point of use (ISP) -- lets it compose
// application/inventory.CreateBatchUseCase/PortionFromBulkUseCase and this
// package's own CreateSKUUseCase without depending on their concrete types.
type batchFinderForBulkAdd interface {
	FindByID(ctx context.Context, id string) (*invDomain.Batch, error)
}
type batchCreatorForBulkAdd interface {
	Execute(ctx context.Context, in invapp.CreateBatchInput) (*invDomain.Batch, error)
}
type portionerForBulkAdd interface {
	Execute(ctx context.Context, in invapp.PortionFromBulkInput) (*invapp.PortionFromBulkResult, error)
}
type skuCreatorForBulkAdd interface {
	Execute(ctx context.Context, in CreateSKUInput) (*pmsDomain.SKU, error)
}

// BulkAddWeightPortionsUseCase is the "type a list of weights" tool: for
// each entered weight, it creates a new sellable SKU (UnitType=unit,
// PricingMode=fixed, PortionedFromSKUID pointing at the bulk pool, priced
// from the bulk's own PricePerKiloMinorUnits, barcoded via
// pms.EncodeWeightEmbeddedBarcode so the checkout scan path can read it
// back), then peels that exact weight off the bulk pool into it (see
// application/inventory.PortionFromBulkUseCase).
//
// Like CompletePickingUseCase's replay, this composes several use cases
// that each open their own transaction (tenant.WithTenant must never
// nest) -- one full create+batch+portion cycle per weight, in order. A
// failure partway through (e.g. weight 4 of 6) leaves the first 3 packages
// already created and stocked; the error names which weight failed so the
// caller can retry just the remainder.
type BulkAddWeightPortionsUseCase struct {
	db          *gorm.DB
	skuRepo     pmsDomain.SKURepository
	batchRepo   batchFinderForBulkAdd
	createSKU   skuCreatorForBulkAdd
	createBatch batchCreatorForBulkAdd
	portion     portionerForBulkAdd
}

func NewBulkAddWeightPortionsUseCase(
	db *gorm.DB, skuRepo pmsDomain.SKURepository, batchRepo batchFinderForBulkAdd,
	createSKU skuCreatorForBulkAdd, createBatch batchCreatorForBulkAdd, portion portionerForBulkAdd,
) *BulkAddWeightPortionsUseCase {
	return &BulkAddWeightPortionsUseCase{
		db: db, skuRepo: skuRepo, batchRepo: batchRepo,
		createSKU: createSKU, createBatch: createBatch, portion: portion,
	}
}

func (uc *BulkAddWeightPortionsUseCase) Execute(ctx context.Context, in BulkAddWeightPortionsInput) (*BulkAddWeightPortionsResult, error) {
	if len(in.WeightsGrams) == 0 {
		return nil, fmt.Errorf("bulkAddWeightPortions: %w", pmsDomain.ErrNoWeightsProvided)
	}
	for _, w := range in.WeightsGrams {
		if w <= 0 {
			return nil, fmt.Errorf("bulkAddWeightPortions: %w", invDomain.ErrInvalidQuantity)
		}
	}

	var bulkSKU pmsDomain.SKU
	var bulkBatch *invDomain.Batch
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByID(ctx, in.BulkSKUID)
		if err != nil {
			return err
		}
		bulkSKU = *sku
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
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("bulkAddWeightPortions: %w", err)
	}

	if bulkSKU.UnitType != pmsDomain.SKUUnitTypeKilo || bulkSKU.PricingMode != pmsDomain.SKUPricingModePerWeight ||
		bulkSKU.PricePerKiloMinorUnits == nil || bulkSKU.ScaleCode == nil {
		return nil, fmt.Errorf("bulkAddWeightPortions: %w", pmsDomain.ErrBulkSKUNotWeightPool)
	}

	codePrefix := in.SKUCodePrefix
	if codePrefix == "" {
		codePrefix = bulkSKU.SKUCode
	}

	result := &BulkAddWeightPortionsResult{Created: make([]pmsDomain.SKU, 0, len(in.WeightsGrams))}
	for _, weight := range in.WeightsGrams {
		created, err := uc.createOnePortion(ctx, in, bulkSKU, bulkBatch, codePrefix, weight)
		if err != nil {
			return nil, fmt.Errorf("bulkAddWeightPortions: package at %dg: %w", weight, err)
		}
		result.Created = append(result.Created, *created)
	}
	return result, nil
}

func (uc *BulkAddWeightPortionsUseCase) createOnePortion(
	ctx context.Context, in BulkAddWeightPortionsInput, bulkSKU pmsDomain.SKU, bulkBatch *invDomain.Batch, codePrefix string, weight int64,
) (*pmsDomain.SKU, error) {
	barcode, err := pmsDomain.EncodeWeightEmbeddedBarcode(*bulkSKU.ScaleCode, weight)
	if err != nil {
		return nil, err
	}
	price := lineTotalForScannedWeight(bulkSKU, weight)
	w := weight

	created, err := uc.createSKU.Execute(ctx, CreateSKUInput{
		VariantID: bulkSKU.VariantID, SKUCode: fmt.Sprintf("%s-%dG", codePrefix, weight), Barcode: barcode,
		PriceMinorUnits: price, CurrencyCode: bulkSKU.CurrencyCode, WeightGrams: &w,
		UnitType: pmsDomain.SKUUnitTypeUnit, PortionedFromSKUID: &bulkSKU.ID,
		PricingMode: pmsDomain.SKUPricingModeFixed, Status: pmsDomain.SKUStatusActive,
	})
	if err != nil {
		return nil, err
	}

	var targetBatchID *string
	if bulkBatch != nil {
		newBatch, err := uc.createBatch.Execute(ctx, invapp.CreateBatchInput{
			SKUID: created.ID, WarehouseID: in.WarehouseID, LotNumber: bulkBatch.LotNumber, ExpiryDate: bulkBatch.ExpiryDate,
		})
		if err != nil {
			return nil, err
		}
		targetBatchID = &newBatch.ID
	}

	if _, err := uc.portion.Execute(ctx, invapp.PortionFromBulkInput{
		TargetSKUID: created.ID, WarehouseID: in.WarehouseID, BatchID: in.BulkBatchID, WeightGrams: &w,
		TargetBatchID: targetBatchID, Reason: in.Reason, OccurredAt: time.Now(),
	}); err != nil {
		return nil, err
	}
	return created, nil
}
