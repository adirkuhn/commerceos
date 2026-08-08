package pms

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateSKUInput struct {
	ID                  string
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
	Status              pmsDomain.SKUStatus
}

type UpdateSKUUseCase struct {
	db          *gorm.DB
	variantRepo pmsDomain.VariantRepository
	skuRepo     pmsDomain.SKURepository
	outboxRepo  outbox.Repository
}

func NewUpdateSKUUseCase(db *gorm.DB, variantRepo pmsDomain.VariantRepository, skuRepo pmsDomain.SKURepository, outboxRepo outbox.Repository) *UpdateSKUUseCase {
	return &UpdateSKUUseCase{db: db, variantRepo: variantRepo, skuRepo: skuRepo, outboxRepo: outboxRepo}
}

// Execute updates price/barcode/currency/status. SKUCode is deliberately
// not editable here — it's the stable external identifier; changing it is
// effectively a different SKU (delete + create).
func (uc *UpdateSKUUseCase) Execute(ctx context.Context, in UpdateSKUInput) (*pmsDomain.SKU, error) {
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

	var sku pmsDomain.SKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.skuRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		if in.CaseOfSKUID != nil {
			if err := validateCaseOfSKU(ctx, uc.skuRepo, in.ID, *in.CaseOfSKUID); err != nil {
				return err
			}
		}
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
		existing.Status = in.Status
		existing.UpdatedAt = time.Now()
		if err := uc.skuRepo.Update(ctx, existing); err != nil {
			return err
		}
		sku = *existing

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
