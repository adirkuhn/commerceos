package pms

import (
	"context"
	"fmt"
	"strings"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateScaleCodeAliasInput struct {
	SKUID string
	Code  string
}

type CreateScaleCodeAliasUseCase struct {
	db        *gorm.DB
	skuRepo   pmsDomain.SKURepository
	aliasRepo pmsDomain.ScaleCodeAliasRepository
}

func NewCreateScaleCodeAliasUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository, aliasRepo pmsDomain.ScaleCodeAliasRepository) *CreateScaleCodeAliasUseCase {
	return &CreateScaleCodeAliasUseCase{db: db, skuRepo: skuRepo, aliasRepo: aliasRepo}
}

func (uc *CreateScaleCodeAliasUseCase) Execute(ctx context.Context, in CreateScaleCodeAliasInput) (*pmsDomain.ScaleCodeAlias, error) {
	in.Code = strings.TrimSpace(in.Code)
	if in.Code == "" {
		return nil, fmt.Errorf("createScaleCodeAlias: %w", pmsDomain.ErrScaleCodeAliasCodeRequired)
	}

	var alias pmsDomain.ScaleCodeAlias
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByID(ctx, in.SKUID)
		if err != nil {
			return err
		}
		now := time.Now()
		alias = pmsDomain.ScaleCodeAlias{
			TenantID: tenant.IDFromContext(ctx), SKUID: sku.ID, Code: in.Code,
			CreatedAt: now, UpdatedAt: now,
		}
		return uc.aliasRepo.Create(ctx, &alias)
	})
	if err != nil {
		return nil, fmt.Errorf("createScaleCodeAlias: %w", err)
	}
	return &alias, nil
}

type ListScaleCodeAliasesUseCase struct {
	db   *gorm.DB
	repo pmsDomain.ScaleCodeAliasRepository
}

func NewListScaleCodeAliasesUseCase(db *gorm.DB, repo pmsDomain.ScaleCodeAliasRepository) *ListScaleCodeAliasesUseCase {
	return &ListScaleCodeAliasesUseCase{db: db, repo: repo}
}

func (uc *ListScaleCodeAliasesUseCase) Execute(ctx context.Context, skuID string) ([]pmsDomain.ScaleCodeAlias, error) {
	var aliases []pmsDomain.ScaleCodeAlias
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		a, err := uc.repo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		aliases = a
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listScaleCodeAliases: %w", err)
	}
	return aliases, nil
}

type DeleteScaleCodeAliasUseCase struct {
	db   *gorm.DB
	repo pmsDomain.ScaleCodeAliasRepository
}

func NewDeleteScaleCodeAliasUseCase(db *gorm.DB, repo pmsDomain.ScaleCodeAliasRepository) *DeleteScaleCodeAliasUseCase {
	return &DeleteScaleCodeAliasUseCase{db: db, repo: repo}
}

func (uc *DeleteScaleCodeAliasUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.repo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteScaleCodeAlias: %w", err)
	}
	return nil
}
