package iam

import (
	"context"
	"fmt"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// GetPairingCodeStatusUseCase backs the desktop's short-poll for whether its
// generated code has been claimed yet -- tenant-scoped (the desktop is
// authenticated), unlike ClaimPairingCodeUseCase which has no tenant
// identity to scope by.
type GetPairingCodeStatusUseCase struct {
	db   *gorm.DB
	repo iamDomain.ScannerPairingCodeRepository
}

func NewGetPairingCodeStatusUseCase(db *gorm.DB, repo iamDomain.ScannerPairingCodeRepository) *GetPairingCodeStatusUseCase {
	return &GetPairingCodeStatusUseCase{db: db, repo: repo}
}

func (uc *GetPairingCodeStatusUseCase) Execute(ctx context.Context, id string) (*iamDomain.ScannerPairingCode, error) {
	var pairing *iamDomain.ScannerPairingCode
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		pairing, err = uc.repo.FindByID(ctx, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getPairingCodeStatus: %w", err)
	}
	return pairing, nil
}
