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

// ListSyncLogsNeedingAttentionUseCase powers the Sync History page's
// "Needs attention" filter -- every listing whose most recent sync attempt
// failed and hasn't been dismissed, newest first. Same
// product-name/store-URL enrichment as ListAllSyncLogsUseCase (see that
// type's own doc for why it's best-effort).
type ListSyncLogsNeedingAttentionUseCase struct {
	db          *gorm.DB
	syncLogRepo channelsDomain.SyncLogRepository
	productRepo pmsDomain.ProductRepository
	connRepo    integrationsDomain.ConnectionRepository
}

func NewListSyncLogsNeedingAttentionUseCase(
	db *gorm.DB,
	syncLogRepo channelsDomain.SyncLogRepository,
	productRepo pmsDomain.ProductRepository,
	connRepo integrationsDomain.ConnectionRepository,
) *ListSyncLogsNeedingAttentionUseCase {
	return &ListSyncLogsNeedingAttentionUseCase{db: db, syncLogRepo: syncLogRepo, productRepo: productRepo, connRepo: connRepo}
}

func (uc *ListSyncLogsNeedingAttentionUseCase) Execute(ctx context.Context, page, perPage int) ([]SyncLogEntryWithNames, int64, error) {
	var entries []channelsDomain.SyncLogEntry
	var total int64
	productNames := map[string]string{}
	storeURLs := map[string]string{}

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		entries, total, err = uc.syncLogRepo.ListNeedingAttention(ctx, page, perPage)
		if err != nil {
			return err
		}
		for _, e := range entries {
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
		return nil, 0, fmt.Errorf("listSyncLogsNeedingAttention: %w", err)
	}

	out := make([]SyncLogEntryWithNames, len(entries))
	for i, e := range entries {
		out[i] = SyncLogEntryWithNames{SyncLogEntry: e, ProductName: productNames[e.ProductID], StoreURL: storeURLs[e.ConnectionID]}
	}
	return out, total, nil
}
