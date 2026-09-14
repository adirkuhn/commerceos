package messagecenter

import (
	"context"
	"fmt"
	"time"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateMessageTemplateInput struct {
	ID       string
	Name     string
	Category mcDomain.Category
	Channels []messaging.Channel
	Subject  string
	Body     string
	Enabled  bool
}

type UpdateMessageTemplateUseCase struct {
	db   *gorm.DB
	repo mcDomain.MessageTemplateRepository
}

func NewUpdateMessageTemplateUseCase(db *gorm.DB, repo mcDomain.MessageTemplateRepository) *UpdateMessageTemplateUseCase {
	return &UpdateMessageTemplateUseCase{db: db, repo: repo}
}

func (uc *UpdateMessageTemplateUseCase) Execute(ctx context.Context, in UpdateMessageTemplateInput) (*mcDomain.MessageTemplate, error) {
	if err := validateMessageTemplateInput(in.Name, in.Category, in.Channels, in.Body); err != nil {
		return nil, fmt.Errorf("updateMessageTemplate: %w", err)
	}

	var tmpl mcDomain.MessageTemplate
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		existing.Name = in.Name
		existing.Category = in.Category
		existing.Channels = in.Channels
		existing.Subject = in.Subject
		existing.Body = in.Body
		existing.Enabled = in.Enabled
		existing.UpdatedAt = time.Now()
		if err := uc.repo.Update(ctx, existing); err != nil {
			return err
		}
		tmpl = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateMessageTemplate: %w", err)
	}
	return &tmpl, nil
}
