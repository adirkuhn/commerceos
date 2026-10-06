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

type CreateScaleCodeAliasInput struct {
	SKUID string
	Code  string
}

type CreateScaleCodeAliasUseCase struct {
	db        *gorm.DB
	skuRepo   pmsDomain.SKURepository
	aliasRepo pmsDomain.ScaleCodeAliasRepository
	outbox    eventrecord.Enqueuer
}

func NewCreateScaleCodeAliasUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository, aliasRepo pmsDomain.ScaleCodeAliasRepository, outbox eventrecord.Enqueuer) *CreateScaleCodeAliasUseCase {
	return &CreateScaleCodeAliasUseCase{db: db, skuRepo: skuRepo, aliasRepo: aliasRepo, outbox: outbox}
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
		if err := uc.aliasRepo.Create(ctx, &alias); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ScaleCodeAliasCreated, alias.ID, eventrecord.Data{"sku_id": alias.SKUID, "code": alias.Code})
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
	db     *gorm.DB
	repo   pmsDomain.ScaleCodeAliasRepository
	outbox eventrecord.Enqueuer
}

func NewDeleteScaleCodeAliasUseCase(db *gorm.DB, repo pmsDomain.ScaleCodeAliasRepository, outbox eventrecord.Enqueuer) *DeleteScaleCodeAliasUseCase {
	return &DeleteScaleCodeAliasUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteScaleCodeAliasUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ScaleCodeAliasDeleted, id)
	})
	if err != nil {
		return fmt.Errorf("deleteScaleCodeAlias: %w", err)
	}
	return nil
}
