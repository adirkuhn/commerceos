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

type UpdateSKUInput struct {
	ID                  string
	SKUCode             string
	Barcode             string
	PriceMinorUnits     int64
	SalePriceMinorUnits *int64
	SaleEndDate         *time.Time
	CurrencyCode        string
	WeightGrams         *int64
	LengthMM            *int64
	WidthMM             *int64
	HeightMM            *int64
	IsVirtual           bool
	TracksExpiry        bool
	CaseOfSKUID         *string
	UnitsPerCase        *int64
	UnitType            pmsDomain.SKUUnitType
	PortionedFromSKUID  *string
	TaxClassID          *string
	Status              pmsDomain.SKUStatus
}

type UpdateSKUUseCase struct {
	db               *gorm.DB
	variantRepo      pmsDomain.VariantRepository
	skuRepo          pmsDomain.SKURepository
	outboxRepo       outbox.Repository
	priceHistoryRepo pmsDomain.SKUPriceHistoryRepository
}

func NewUpdateSKUUseCase(db *gorm.DB, variantRepo pmsDomain.VariantRepository, skuRepo pmsDomain.SKURepository, outboxRepo outbox.Repository, priceHistoryRepo pmsDomain.SKUPriceHistoryRepository) *UpdateSKUUseCase {
	return &UpdateSKUUseCase{db: db, variantRepo: variantRepo, skuRepo: skuRepo, outboxRepo: outboxRepo, priceHistoryRepo: priceHistoryRepo}
}

// Execute updates every editable SKU field, including SKUCode -- guarded
// by the same per-tenant uniqueness check CreateSKUUseCase applies
// (ErrDuplicateSKUCode), skipped when the code is unchanged so a SKU
// re-saving its own current code doesn't collide with itself.
func (uc *UpdateSKUUseCase) Execute(ctx context.Context, in UpdateSKUInput) (*pmsDomain.SKU, error) {
	if in.UnitType == "" {
		in.UnitType = pmsDomain.SKUUnitTypeUnit
	}
	// See CreateSKUUseCase's identical trim for why.
	in.Barcode = strings.TrimSpace(in.Barcode)
	if err := validateVirtualDimensions(in.IsVirtual, in.WeightGrams, in.LengthMM, in.WidthMM, in.HeightMM); err != nil {
		return nil, fmt.Errorf("updateSKU: %w", err)
	}
	if err := validateSalePrice(in.SalePriceMinorUnits, in.PriceMinorUnits); err != nil {
		return nil, fmt.Errorf("updateSKU: %w", err)
	}
	if err := validateSaleEndDate(in.SaleEndDate, in.SalePriceMinorUnits); err != nil {
		return nil, fmt.Errorf("updateSKU: %w", err)
	}
	if err := validateCasePackShape(in.CaseOfSKUID, in.UnitsPerCase); err != nil {
		return nil, fmt.Errorf("updateSKU: %w", err)
	}
	if err := validateSKUUnitType(in.UnitType); err != nil {
		return nil, fmt.Errorf("updateSKU: %w", err)
	}
	if err := validatePortionedFromShape(in.UnitType, in.PortionedFromSKUID); err != nil {
		return nil, fmt.Errorf("updateSKU: %w", err)
	}

	var sku pmsDomain.SKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.skuRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		if in.SKUCode != existing.SKUCode {
			other, err := uc.skuRepo.FindByCode(ctx, in.SKUCode)
			if err == nil && other.ID != in.ID {
				return pmsDomain.ErrDuplicateSKUCode
			}
			if err != nil && !errors.Is(err, pmsDomain.ErrSKUNotFound) {
				return err
			}
		}
		if in.CaseOfSKUID != nil {
			if err := validateCaseOfSKU(ctx, uc.skuRepo, in.ID, *in.CaseOfSKUID); err != nil {
				return err
			}
		}
		if in.PortionedFromSKUID != nil {
			if err := validatePortionedFromSKU(ctx, uc.skuRepo, in.ID, *in.PortionedFromSKUID); err != nil {
				return err
			}
		}
		oldPriceMinorUnits := existing.PriceMinorUnits
		oldSalePriceMinorUnits := existing.SalePriceMinorUnits

		existing.SKUCode = in.SKUCode
		existing.Barcode = in.Barcode
		existing.PriceMinorUnits = in.PriceMinorUnits
		existing.SalePriceMinorUnits = in.SalePriceMinorUnits
		existing.SaleEndDate = in.SaleEndDate
		existing.CurrencyCode = in.CurrencyCode
		existing.WeightGrams = in.WeightGrams
		existing.LengthMM = in.LengthMM
		existing.WidthMM = in.WidthMM
		existing.HeightMM = in.HeightMM
		existing.IsVirtual = in.IsVirtual
		existing.TracksExpiry = in.TracksExpiry
		existing.CaseOfSKUID = in.CaseOfSKUID
		existing.UnitsPerCase = in.UnitsPerCase
		existing.UnitType = in.UnitType
		existing.PortionedFromSKUID = in.PortionedFromSKUID
		existing.TaxClassID = in.TaxClassID
		existing.Status = in.Status
		existing.UpdatedAt = time.Now()
		if err := uc.skuRepo.Update(ctx, existing); err != nil {
			return err
		}
		sku = *existing

		if err := recordSKUPriceChangeIfNeeded(ctx, uc.priceHistoryRepo, &sku, oldPriceMinorUnits, oldSalePriceMinorUnits, pmsDomain.PriceChangeReasonManualUpdate); err != nil {
			return err
		}

		variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID)
		if err != nil {
			return err
		}
		return enqueueProductChanged(ctx, uc.outboxRepo, variant.ProductID)
	})
	if err != nil {
		return nil, fmt.Errorf("updateSKU: %w", err)
	}
	return &sku, nil
}
