package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/events"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// EventEnqueuer is the one outbox operation a settings write needs
// (satisfied by outbox.Repository).
type EventEnqueuer interface {
	Enqueue(ctx context.Context, subject, eventID string, payload []byte, occurredAt time.Time) error
}

type UpdateCompanyProfileUseCase struct {
	db     *gorm.DB
	repo   settingsDomain.CompanyProfileRepository
	outbox EventEnqueuer
}

func NewUpdateCompanyProfileUseCase(db *gorm.DB, repo settingsDomain.CompanyProfileRepository, outbox EventEnqueuer) *UpdateCompanyProfileUseCase {
	return &UpdateCompanyProfileUseCase{db: db, repo: repo, outbox: outbox}
}

// Execute replaces the tenant's company profile and raises
// CompanyProfileUpdated in the same transaction.
func (uc *UpdateCompanyProfileUseCase) Execute(ctx context.Context, in settingsDomain.CompanyProfileInput) (*settingsDomain.CompanyProfile, error) {
	var result *settingsDomain.CompanyProfile
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now().UTC()
		tenantID := tenant.IDFromContext(ctx)
		p, err := settingsDomain.NewCompanyProfile(tenantID, in, now)
		if err != nil {
			return err
		}
		if err := uc.repo.Upsert(ctx, p); err != nil {
			return err
		}
		event := events.CompanyProfileUpdated{
			Base:     events.Base{EventID: idgen.NewV4(), OccurredAt: now},
			TenantID: tenantID,
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		if err := uc.outbox.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt); err != nil {
			return err
		}
		result = p
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateCompanyProfile: %w", err)
	}
	return result, nil
}
