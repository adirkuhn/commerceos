package pms

import (
	"context"
	"errors"
	"fmt"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateSKUInput struct {
	VariantID       string
	SKUCode         string
	Barcode         string
	PriceMinorUnits int64
	CurrencyCode    string
	WeightGrams     *int64
	LengthMM        *int64
	WidthMM         *int64
	HeightMM        *int64
	IsVirtual       bool
	TracksExpiry    bool
	Status          pmsDomain.SKUStatus
}

type CreateSKUUseCase struct {
	db          *gorm.DB
	variantRepo pmsDomain.VariantRepository
	skuRepo     pmsDomain.SKURepository
}

func NewCreateSKUUseCase(db *gorm.DB, variantRepo pmsDomain.VariantRepository, skuRepo pmsDomain.SKURepository) *CreateSKUUseCase {
	return &CreateSKUUseCase{db: db, variantRepo: variantRepo, skuRepo: skuRepo}
}

func (uc *CreateSKUUseCase) Execute(ctx context.Context, in CreateSKUInput) (*pmsDomain.SKU, error) {
	if err := validateVirtualDimensions(in.IsVirtual, in.WeightGrams, in.LengthMM, in.WidthMM, in.HeightMM); err != nil {
		return nil, fmt.Errorf("createSKU: %w", err)
	}

	var sku pmsDomain.SKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.variantRepo.FindByID(ctx, in.VariantID); err != nil {
			return err
		}

		_, err := uc.skuRepo.FindByCode(ctx, in.SKUCode)
		if err == nil {
			return pmsDomain.ErrDuplicateSKUCode
		}
		if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return err
		}

		now := time.Now()
		sku = pmsDomain.SKU{
			TenantID:        tenant.IDFromContext(ctx),
			VariantID:       in.VariantID,
			SKUCode:         in.SKUCode,
			Barcode:         in.Barcode,
			PriceMinorUnits: in.PriceMinorUnits,
			CurrencyCode:    in.CurrencyCode,
			WeightGrams:     in.WeightGrams,
			LengthMM:        in.LengthMM,
			WidthMM:         in.WidthMM,
			HeightMM:        in.HeightMM,
			IsVirtual:       in.IsVirtual,
			TracksExpiry:    in.TracksExpiry,
			Status:          in.Status,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		return uc.skuRepo.Create(ctx, &sku)
	})
	if err != nil {
		return nil, fmt.Errorf("createSKU: %w", err)
	}
	return &sku, nil
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
