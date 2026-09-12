package pms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateSKUInput struct {
	VariantID              string
	SKUCode                string
	Barcode                string
	PriceMinorUnits        int64
	SalePriceMinorUnits    *int64
	SaleEndDate            *time.Time
	CurrencyCode           string
	WeightGrams            *int64
	LengthMM               *int64
	WidthMM                *int64
	HeightMM               *int64
	IsVirtual              bool
	TracksExpiry           bool
	CaseOfSKUID            *string
	UnitsPerCase           *int64
	UnitType               pmsDomain.SKUUnitType
	PortionedFromSKUID     *string
	ScaleCode              *string
	PricingMode            pmsDomain.SKUPricingMode
	PricePerKiloMinorUnits *int64
	TaxClassID             *string
	Status                 pmsDomain.SKUStatus
}

type CreateSKUUseCase struct {
	db               *gorm.DB
	variantRepo      pmsDomain.VariantRepository
	skuRepo          pmsDomain.SKURepository
	priceHistoryRepo pmsDomain.SKUPriceHistoryRepository
	outboxRepo       outbox.Repository
}

func NewCreateSKUUseCase(db *gorm.DB, variantRepo pmsDomain.VariantRepository, skuRepo pmsDomain.SKURepository, priceHistoryRepo pmsDomain.SKUPriceHistoryRepository, outboxRepo outbox.Repository) *CreateSKUUseCase {
	return &CreateSKUUseCase{db: db, variantRepo: variantRepo, skuRepo: skuRepo, priceHistoryRepo: priceHistoryRepo, outboxRepo: outboxRepo}
}

func (uc *CreateSKUUseCase) Execute(ctx context.Context, in CreateSKUInput) (*pmsDomain.SKU, error) {
	if in.UnitType == "" {
		in.UnitType = pmsDomain.SKUUnitTypeUnit
	}
	// A scanner, copy-paste, or manual entry can leave stray whitespace
	// around an otherwise-correct barcode -- trim at write time so every
	// later exact-match lookup (SKURepository.FindByBarcode) actually
	// works, rather than silently failing to find the SKU.
	in.Barcode = strings.TrimSpace(in.Barcode)
	if err := validateVirtualDimensions(in.IsVirtual, in.WeightGrams, in.LengthMM, in.WidthMM, in.HeightMM); err != nil {
		return nil, fmt.Errorf("createSKU: %w", err)
	}
	if err := validateSalePrice(in.SalePriceMinorUnits, in.PriceMinorUnits); err != nil {
		return nil, fmt.Errorf("createSKU: %w", err)
	}
	if err := validateSaleEndDate(in.SaleEndDate, in.SalePriceMinorUnits); err != nil {
		return nil, fmt.Errorf("createSKU: %w", err)
	}
	if err := validateCasePackShape(in.CaseOfSKUID, in.UnitsPerCase); err != nil {
		return nil, fmt.Errorf("createSKU: %w", err)
	}
	if err := validateSKUUnitType(in.UnitType); err != nil {
		return nil, fmt.Errorf("createSKU: %w", err)
	}
	if err := validatePortionedFromShape(in.UnitType, in.PortionedFromSKUID); err != nil {
		return nil, fmt.Errorf("createSKU: %w", err)
	}
	if in.PricingMode == "" {
		in.PricingMode = pmsDomain.SKUPricingModeFixed
	}
	if err := validateSKUPricingMode(in.PricingMode, in.PricePerKiloMinorUnits); err != nil {
		return nil, fmt.Errorf("createSKU: %w", err)
	}

	var sku pmsDomain.SKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		variant, err := uc.variantRepo.FindByID(ctx, in.VariantID)
		if err != nil {
			return err
		}

		_, err = uc.skuRepo.FindByCode(ctx, in.SKUCode)
		if err == nil {
			return pmsDomain.ErrDuplicateSKUCode
		}
		if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return err
		}

		if in.CaseOfSKUID != nil {
			if err := validateCaseOfSKU(ctx, uc.skuRepo, "", *in.CaseOfSKUID); err != nil {
				return err
			}
		}
		if in.PortionedFromSKUID != nil {
			if err := validatePortionedFromSKU(ctx, uc.skuRepo, "", *in.PortionedFromSKUID); err != nil {
				return err
			}
		}

		now := time.Now()
		sku = pmsDomain.SKU{
			TenantID:               tenant.IDFromContext(ctx),
			VariantID:              in.VariantID,
			SKUCode:                in.SKUCode,
			Barcode:                in.Barcode,
			PriceMinorUnits:        in.PriceMinorUnits,
			SalePriceMinorUnits:    in.SalePriceMinorUnits,
			SaleEndDate:            in.SaleEndDate,
			CurrencyCode:           in.CurrencyCode,
			WeightGrams:            in.WeightGrams,
			LengthMM:               in.LengthMM,
			WidthMM:                in.WidthMM,
			HeightMM:               in.HeightMM,
			IsVirtual:              in.IsVirtual,
			TracksExpiry:           in.TracksExpiry,
			CaseOfSKUID:            in.CaseOfSKUID,
			UnitsPerCase:           in.UnitsPerCase,
			UnitType:               in.UnitType,
			PortionedFromSKUID:     in.PortionedFromSKUID,
			ScaleCode:              in.ScaleCode,
			PricingMode:            in.PricingMode,
			PricePerKiloMinorUnits: in.PricePerKiloMinorUnits,
			TaxClassID:             in.TaxClassID,
			Status:                 in.Status,
			CreatedAt:              now,
			UpdatedAt:              now,
		}
		if err := uc.skuRepo.Create(ctx, &sku); err != nil {
			return err
		}
		if err := writeSKUPriceHistory(ctx, uc.priceHistoryRepo, &sku, pmsDomain.PriceChangeReasonCreated); err != nil {
			return err
		}
		// A brand-new variant-mode product's very first enrollment attempt
		// (fired by CreateProductUseCase's own pms.product.changed, right
		// when the product row is created) finds zero variants and bails
		// out without ever creating a ProductChannelListing -- this SKU
		// existing now is exactly what that attempt was missing, so it must
		// re-trigger enrollment itself rather than leave the product
		// permanently unsynced until some unrelated later edit happens to.
		return enqueueProductChanged(ctx, uc.outboxRepo, variant.ProductID)
	})
	if err != nil {
		return nil, fmt.Errorf("createSKU: %w", err)
	}
	return &sku, nil
}

// validateSalePrice is the shared invariant for CreateSKU/UpdateSKU: a
// sale price, if set, can't exceed the regular price.
func validateSalePrice(salePriceMinorUnits *int64, priceMinorUnits int64) error {
	if salePriceMinorUnits == nil {
		return nil
	}
	if *salePriceMinorUnits > priceMinorUnits {
		return pmsDomain.ErrSalePriceExceedsPrice
	}
	return nil
}

// validateSaleEndDate is the shared invariant for CreateSKU/UpdateSKU: a
// sale end date only means something alongside an actual sale price.
func validateSaleEndDate(saleEndDate *time.Time, salePriceMinorUnits *int64) error {
	if saleEndDate != nil && salePriceMinorUnits == nil {
		return pmsDomain.ErrSaleEndDateRequiresSalePrice
	}
	return nil
}

// validateVirtualDimensions is the split-config invariant shared by
// CreateSKU/UpdateSKU: a virtual (digital) SKU never ships, so it can never
// carry weight/dimensions — a physical SKU may set any of them.
func validateVirtualDimensions(isVirtual bool, weightGrams, lengthMM, widthMM, heightMM *int64) error {
	if !isVirtual {
		return nil
	}
	if weightGrams != nil || lengthMM != nil || widthMM != nil || heightMM != nil {
		return pmsDomain.ErrVirtualSKUCannotHaveDimensions
	}
	return nil
}

// validateCasePackShape is the shared invariant for CreateSKU/UpdateSKU:
// CaseOfSKUID and UnitsPerCase are set together or not at all, and
// UnitsPerCase (when set) must be positive.
func validateCasePackShape(caseOfSKUID *string, unitsPerCase *int64) error {
	if (caseOfSKUID == nil) != (unitsPerCase == nil) {
		return pmsDomain.ErrCaseOfSKURequiresUnitsPerCase
	}
	if unitsPerCase != nil && *unitsPerCase <= 0 {
		return pmsDomain.ErrUnitsPerCaseMustBePositive
	}
	return nil
}

// validateCaseOfSKU is the shared invariant for CreateSKU/UpdateSKU once
// CaseOfSKUID is non-nil: the referenced SKU must exist (ErrSKUNotFound
// propagates as-is), can't be the SKU itself (selfID is "" for a not-yet-
// created SKU, which can never equal a real caseOfSKUID), and can't itself
// already be a case of another SKU (see ErrCaseOfSKUCannotChain -- keeps
// the relationship one level deep).
func validateCaseOfSKU(ctx context.Context, skuRepo pmsDomain.SKURepository, selfID, caseOfSKUID string) error {
	if selfID != "" && selfID == caseOfSKUID {
		return pmsDomain.ErrCaseOfSKUCannotBeSelf
	}
	target, err := skuRepo.FindByID(ctx, caseOfSKUID)
	if err != nil {
		return err
	}
	if target.CaseOfSKUID != nil {
		return pmsDomain.ErrCaseOfSKUCannotChain
	}
	return nil
}

// validateSKUUnitType guards SKU.UnitType against any value outside the
// three declared constants.
func validateSKUUnitType(unitType pmsDomain.SKUUnitType) error {
	switch unitType {
	case pmsDomain.SKUUnitTypeUnit, pmsDomain.SKUUnitTypeBox, pmsDomain.SKUUnitTypeKilo:
		return nil
	default:
		return pmsDomain.ErrInvalidSKUUnitType
	}
}

// validateSKUPricingMode is the shared invariant for CreateSKU/UpdateSKU:
// PricingMode must be a declared constant, PricePerKiloMinorUnits must be
// set (and positive) exactly when PricingMode is per_weight, and must be
// nil for fixed pricing (the default) -- see SKU.PricingMode's doc
// comment for why the two are orthogonal to UnitType.
func validateSKUPricingMode(pricingMode pmsDomain.SKUPricingMode, pricePerKiloMinorUnits *int64) error {
	switch pricingMode {
	case pmsDomain.SKUPricingModeFixed:
		if pricePerKiloMinorUnits != nil {
			return pmsDomain.ErrPricePerKiloOnlyForPerWeight
		}
	case pmsDomain.SKUPricingModePerWeight:
		if pricePerKiloMinorUnits == nil {
			return pmsDomain.ErrPricePerKiloRequiredForPerWeight
		}
		if *pricePerKiloMinorUnits <= 0 {
			return pmsDomain.ErrInvalidPricePerKilo
		}
	default:
		return pmsDomain.ErrInvalidSKUPricingMode
	}
	return nil
}

// validatePortionedFromShape is the shared invariant for CreateSKU/UpdateSKU:
// a SKU can't be both a bulk weight pool (UnitType=kilo) and itself
// portioned from another one -- keeps the relationship one level deep
// without needing to walk a chain.
func validatePortionedFromShape(unitType pmsDomain.SKUUnitType, portionedFromSKUID *string) error {
	if unitType == pmsDomain.SKUUnitTypeKilo && portionedFromSKUID != nil {
		return pmsDomain.ErrBulkSKUCannotBePortioned
	}
	return nil
}

// validatePortionedFromSKU is the shared invariant for CreateSKU/UpdateSKU
// once PortionedFromSKUID is non-nil: the referenced SKU must exist
// (ErrSKUNotFound propagates as-is), can't be the SKU itself (selfID is ""
// for a not-yet-created SKU, which can never equal a real
// portionedFromSKUID), and must be a weight pool (UnitType=kilo) --
// validatePortionedFromShape already guarantees a kilo SKU never has its
// own PortionedFromSKUID set, so this alone keeps the relationship one
// level deep.
func validatePortionedFromSKU(ctx context.Context, skuRepo pmsDomain.SKURepository, selfID, portionedFromSKUID string) error {
	if selfID != "" && selfID == portionedFromSKUID {
		return pmsDomain.ErrPortionedFromSKUCannotBeSelf
	}
	target, err := skuRepo.FindByID(ctx, portionedFromSKUID)
	if err != nil {
		return err
	}
	if target.UnitType != pmsDomain.SKUUnitTypeKilo {
		return pmsDomain.ErrPortionedFromSKUMustTrackWeight
	}
	return nil
}
