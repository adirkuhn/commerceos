package pms

import (
	"context"
	"fmt"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateSKUInput struct {
	ID              string
	Barcode         string
	PriceMinorUnits int64
	CurrencyCode    string
	WeightGrams     *int64
	LengthMM        *int64
	WidthMM         *int64
	HeightMM        *int64
	IsVirtual       bool
	Status          pmsDomain.SKUStatus
}

type UpdateSKUUseCase struct {
	db      *gorm.DB
	skuRepo pmsDomain.SKURepository
}

func NewUpdateSKUUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository) *UpdateSKUUseCase {
	return &UpdateSKUUseCase{db: db, skuRepo: skuRepo}
}

// Execute updates price/barcode/currency/status. SKUCode is deliberately
// not editable here — it's the stable external identifier; changing it is
// effectively a different SKU (delete + create).
func (uc *UpdateSKUUseCase) Execute(ctx context.Context, in UpdateSKUInput) (*pmsDomain.SKU, error) {
	if err := validateVirtualDimensions(in.IsVirtual, in.WeightGrams, in.LengthMM, in.WidthMM, in.HeightMM); err != nil {
		return nil, fmt.Errorf("updateSKU: %w", err)
	}

	var sku pmsDomain.SKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.skuRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		existing.Barcode = in.Barcode
		existing.PriceMinorUnits = in.PriceMinorUnits
		existing.CurrencyCode = in.CurrencyCode
		existing.WeightGrams = in.WeightGrams
		existing.LengthMM = in.LengthMM
		existing.WidthMM = in.WidthMM
		existing.HeightMM = in.HeightMM
		existing.IsVirtual = in.IsVirtual
		existing.Status = in.Status
		existing.UpdatedAt = time.Now()
		if err := uc.skuRepo.Update(ctx, existing); err != nil {
			return err
		}
		sku = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateSKU: %w", err)
	}
	return &sku, nil
}
