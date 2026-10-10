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

type DeleteAutomationRuleUseCase struct {
	db     *gorm.DB
	repo   mcDomain.AutomationRuleRepository
	outbox eventrecord.Enqueuer
}

func NewDeleteAutomationRuleUseCase(db *gorm.DB, repo mcDomain.AutomationRuleRepository, outbox eventrecord.Enqueuer) *DeleteAutomationRuleUseCase {
	return &DeleteAutomationRuleUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteAutomationRuleUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.repo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.AutomationRuleDeleted, id, eventrecord.Data{"name": existing.Name})
	})
	if err != nil {
		return fmt.Errorf("deleteAutomationRule: %w", err)
	}
	return nil
}
