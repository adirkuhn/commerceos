package pms

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateBarcodeAliasInput struct {
	SKUID string
	Code  string
	// UnitsPerScan defaults to 1 (a plain alternate unit barcode) when
	// unset -- pass > 1 for a case/box multiplier.
	UnitsPerScan int64
}

type CreateBarcodeAliasUseCase struct {
	db        *gorm.DB
	skuRepo   pmsDomain.SKURepository
	aliasRepo pmsDomain.BarcodeAliasRepository
	outbox    eventrecord.Enqueuer
}

func NewCreateBarcodeAliasUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository, aliasRepo pmsDomain.BarcodeAliasRepository, outbox eventrecord.Enqueuer) *CreateBarcodeAliasUseCase {
	return &CreateBarcodeAliasUseCase{db: db, skuRepo: skuRepo, aliasRepo: aliasRepo, outbox: outbox}
}

func (uc *CreateBarcodeAliasUseCase) Execute(ctx context.Context, in CreateBarcodeAliasInput) (*pmsDomain.BarcodeAlias, error) {
	in.Code = strings.TrimSpace(in.Code)
	if in.Code == "" {
		return nil, fmt.Errorf("createBarcodeAlias: %w", pmsDomain.ErrBarcodeAliasCodeRequired)
	}
	if in.UnitsPerScan == 0 {
		in.UnitsPerScan = 1
	}
	if in.UnitsPerScan < 1 {
		return nil, fmt.Errorf("createBarcodeAlias: %w", pmsDomain.ErrInvalidUnitsPerScan)
	}

	var alias pmsDomain.BarcodeAlias
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByID(ctx, in.SKUID)
		if err != nil {
			return err
		}
		now := time.Now()
		alias = pmsDomain.BarcodeAlias{
			TenantID: tenant.IDFromContext(ctx), SKUID: sku.ID, Code: in.Code, UnitsPerScan: in.UnitsPerScan,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.aliasRepo.Create(ctx, &alias); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.BarcodeAliasCreated, alias.ID, eventrecord.Data{"sku_id": alias.SKUID, "code": alias.Code, "units_per_scan": alias.UnitsPerScan})
	})
	if err != nil {
		return nil, fmt.Errorf("createBarcodeAlias: %w", err)
	}
	return &alias, nil
}

type ListBarcodeAliasesUseCase struct {
	db   *gorm.DB
	repo pmsDomain.BarcodeAliasRepository
}

func NewListBarcodeAliasesUseCase(db *gorm.DB, repo pmsDomain.BarcodeAliasRepository) *ListBarcodeAliasesUseCase {
	return &ListBarcodeAliasesUseCase{db: db, repo: repo}
}

func (uc *ListBarcodeAliasesUseCase) Execute(ctx context.Context, skuID string) ([]pmsDomain.BarcodeAlias, error) {
	var aliases []pmsDomain.BarcodeAlias
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		a, err := uc.repo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		aliases = a
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listBarcodeAliases: %w", err)
	}
	return aliases, nil
}

type DeleteBarcodeAliasUseCase struct {
	db     *gorm.DB
	repo   pmsDomain.BarcodeAliasRepository
	outbox eventrecord.Enqueuer
}

func NewDeleteBarcodeAliasUseCase(db *gorm.DB, repo pmsDomain.BarcodeAliasRepository, outbox eventrecord.Enqueuer) *DeleteBarcodeAliasUseCase {
	return &DeleteBarcodeAliasUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteBarcodeAliasUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.BarcodeAliasDeleted, id)
	})
	if err != nil {
		return fmt.Errorf("deleteBarcodeAlias: %w", err)
	}
	return nil
}
