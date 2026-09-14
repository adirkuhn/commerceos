package messagecenter

import (
	"context"
	"fmt"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListMessageTemplatesUseCase struct {
	db   *gorm.DB
	repo mcDomain.MessageTemplateRepository
}

func NewListMessageTemplatesUseCase(db *gorm.DB, repo mcDomain.MessageTemplateRepository) *ListMessageTemplatesUseCase {
	return &ListMessageTemplatesUseCase{db: db, repo: repo}
}

func (uc *ListMessageTemplatesUseCase) Execute(ctx context.Context) ([]mcDomain.MessageTemplate, error) {
	var templates []mcDomain.MessageTemplate
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		templates, err = uc.repo.ListByTenant(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listMessageTemplates: %w", err)
	}
	return templates, nil
}
