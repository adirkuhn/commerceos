package messagecenter

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteMessageTemplateUseCase struct {
	db     *gorm.DB
	repo   mcDomain.MessageTemplateRepository
	outbox eventrecord.Enqueuer
}

func NewDeleteMessageTemplateUseCase(db *gorm.DB, repo mcDomain.MessageTemplateRepository, outbox eventrecord.Enqueuer) *DeleteMessageTemplateUseCase {
	return &DeleteMessageTemplateUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteMessageTemplateUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.repo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.MessageTemplateDeleted, id, eventrecord.Data{"name": existing.Name})
	})
	if err != nil {
		return fmt.Errorf("deleteMessageTemplate: %w", err)
	}
	return nil
}
