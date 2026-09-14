package messagecenter

import (
	"context"
	"fmt"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteMessageTemplateUseCase struct {
	db   *gorm.DB
	repo mcDomain.MessageTemplateRepository
}

func NewDeleteMessageTemplateUseCase(db *gorm.DB, repo mcDomain.MessageTemplateRepository) *DeleteMessageTemplateUseCase {
	return &DeleteMessageTemplateUseCase{db: db, repo: repo}
}

func (uc *DeleteMessageTemplateUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.repo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.repo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteMessageTemplate: %w", err)
	}
	return nil
}
