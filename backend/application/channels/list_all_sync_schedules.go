package channels

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// connectionLister is integrations.ConnectionRepository narrowed to point
// of use (ISP) -- ListByTenant has no WHERE tenant_id of its own, so under
// bypass_rls (tenant.WithSuperAdmin) it returns every tenant's connections,
// same convention cmd/repair-stock-review-flags and
// cmd/backfill-woo-sale-prices already rely on.
type connectionLister interface {
	ListByTenant(ctx context.Context) ([]integrationsDomain.Connection, error)
}

// SyncScheduleWithConnection pairs a SyncSchedule with its connection's
// StoreURL -- a raw ConnectionID/TenantID means nothing to a human reading
// the super_admin "Scheduled Jobs" admin page, the store URL does.
type SyncScheduleWithConnection struct {
	Schedule           channelsDomain.SyncSchedule
	ConnectionStoreURL string
}

// ListAllSyncSchedulesUseCase is the "Scheduled Jobs" admin page's read
// side for the per-tenant half of the page (Woo sync schedules) --
// cross-tenant, every connection's schedules in one list, enriched with
// each one's store URL so a super_admin can tell them apart.
type ListAllSyncSchedulesUseCase struct {
	db           *gorm.DB
	scheduleRepo channelsDomain.SyncScheduleRepository
	connRepo     connectionLister
}

func NewListAllSyncSchedulesUseCase(db *gorm.DB, scheduleRepo channelsDomain.SyncScheduleRepository, connRepo connectionLister) *ListAllSyncSchedulesUseCase {
	return &ListAllSyncSchedulesUseCase{db: db, scheduleRepo: scheduleRepo, connRepo: connRepo}
}

func (uc *ListAllSyncSchedulesUseCase) Execute(ctx context.Context) ([]SyncScheduleWithConnection, error) {
	adminCtx := tenant.WithSuperAdmin(ctx)
	var schedules []channelsDomain.SyncSchedule
	var conns []integrationsDomain.Connection
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		var err error
		if schedules, err = uc.scheduleRepo.ListAll(ctx); err != nil {
			return err
		}
		conns, err = uc.connRepo.ListByTenant(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listAllSyncSchedules: %w", err)
	}

	storeURLByConn := make(map[string]string, len(conns))
	for _, c := range conns {
		storeURLByConn[c.ID] = string(c.StoreURL)
	}
	out := make([]SyncScheduleWithConnection, len(schedules))
	for i, s := range schedules {
		out[i] = SyncScheduleWithConnection{Schedule: s, ConnectionStoreURL: storeURLByConn[s.ConnectionID]}
	}
	return out, nil
}
