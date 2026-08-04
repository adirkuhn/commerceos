package channels

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// SyncLogEntryWithNames is a SyncLogEntry enriched with the human-readable
// product name and store URL a tenant-wide history page needs to display —
// ListByListing's callers already know the product/connection from
// context, so only this "list everything" path needs the join.
type SyncLogEntryWithNames struct {
	channelsDomain.SyncLogEntry
	ProductName string
	StoreURL    string
}

// ListAllSyncLogsUseCase powers the tenant-wide sync history page — every
// push/import/stock attempt across every product and connection, newest
// first.
type ListAllSyncLogsUseCase struct {
	db          *gorm.DB
	syncLogRepo channelsDomain.SyncLogRepository
	productRepo pmsDomain.ProductRepository
	connRepo    integrationsDomain.ConnectionRepository
}

func NewListAllSyncLogsUseCase(
	db *gorm.DB,
	syncLogRepo channelsDomain.SyncLogRepository,
	productRepo pmsDomain.ProductRepository,
	connRepo integrationsDomain.ConnectionRepository,
) *ListAllSyncLogsUseCase {
	return &ListAllSyncLogsUseCase{db: db, syncLogRepo: syncLogRepo, productRepo: productRepo, connRepo: connRepo}
}

func (uc *ListAllSyncLogsUseCase) Execute(ctx context.Context, page, perPage int) ([]SyncLogEntryWithNames, int64, error) {
	var entries []channelsDomain.SyncLogEntry
	var total int64
	productNames := map[string]string{}
	storeURLs := map[string]string{}

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		entries, total, err = uc.syncLogRepo.ListAll(ctx, page, perPage)
		if err != nil {
			return err
		}
		for _, e := range entries {
			// Best-effort enrichment: a product/connection deleted since
			// the sync ran just leaves the name blank, never fails the
			// whole page over one stale reference.
			if _, ok := productNames[e.ProductID]; !ok {
				if p, err := uc.productRepo.FindByID(ctx, e.ProductID); err == nil {
					productNames[e.ProductID] = p.Name
				}
			}
			if _, ok := storeURLs[e.ConnectionID]; !ok {
				if c, err := uc.connRepo.FindByID(ctx, e.ConnectionID); err == nil {
					storeURLs[e.ConnectionID] = c.StoreURL.String()
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("listAllSyncLogs: %w", err)
	}

	out := make([]SyncLogEntryWithNames, len(entries))
	for i, e := range entries {
		out[i] = SyncLogEntryWithNames{SyncLogEntry: e, ProductName: productNames[e.ProductID], StoreURL: storeURLs[e.ConnectionID]}
	}
	return out, total, nil
}
