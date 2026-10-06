package channels

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// notifier is the one method of application/notifications'
// CreateNotificationUseCase this use case needs (ISP). Called as a separate
// step *after* the job-creation transaction commits — never nested inside
// it, for the same reason import_woo_products.go's stockRecorder isn't:
// tenant.WithTenant opens a real transaction on the shared *gorm.DB, and
// calling another use case's Execute (which calls tenant.WithTenant again)
// from inside an outer transaction's closure would silently open a second,
// unrelated transaction rather than nesting.
type notifier interface {
	Execute(ctx context.Context, in appnotifications.CreateNotificationInput) (*notificationsDomain.Notification, error)
}

type CreateImportJobInput struct {
	ConnectionID       string
	DefaultWarehouseID string
	DefaultCurrency    string
}

// CreateImportJobUseCase enqueues a one-off Woo import instead of running it
// inline — the old synchronous handler blocked the request (and the
// browser tab) for as long as the import took, which for a real catalog
// (hundreds of products, dozens of WooCommerce API round trips) could be
// minutes. cmd/scheduler's poller picks up pending jobs and runs the same
// ImportWooProductsUseCase this replaces at the handler level.
type CreateImportJobUseCase struct {
	db            *gorm.DB
	importJobRepo channelsDomain.ImportJobRepository
	listingRepo   channelsDomain.ProductChannelListingRepository
	notifier      notifier
	outbox        eventrecord.Enqueuer
}

func NewCreateImportJobUseCase(
	db *gorm.DB, importJobRepo channelsDomain.ImportJobRepository,
	listingRepo channelsDomain.ProductChannelListingRepository, notifier notifier,
	outbox eventrecord.Enqueuer,
) *CreateImportJobUseCase {
	return &CreateImportJobUseCase{db: db, importJobRepo: importJobRepo, listingRepo: listingRepo, notifier: notifier, outbox: outbox}
}

func (uc *CreateImportJobUseCase) Execute(ctx context.Context, in CreateImportJobInput) (*channelsDomain.ImportJob, error) {
	var job channelsDomain.ImportJob
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		active, err := uc.importJobRepo.HasActiveForConnection(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		if active {
			return channelsDomain.ErrImportJobAlreadyActive
		}

		// Decided once, here, before any attempt at this job ever runs --
		// and stuck to across retries via the persisted column, since
		// re-deriving it dynamically at run time would break the moment a
		// crash-and-requeue happens mid-first-import (some products would
		// already exist by the retry, wrongly reading as "not first"). See
		// RunManualWooImportUseCase.
		existingListings, err := uc.listingRepo.ListByConnection(ctx, in.ConnectionID)
		if err != nil {
			return err
		}

		job = channelsDomain.ImportJob{
			TenantID:           tenant.IDFromContext(ctx),
			ConnectionID:       in.ConnectionID,
			DefaultWarehouseID: in.DefaultWarehouseID,
			DefaultCurrency:    in.DefaultCurrency,
			Status:             channelsDomain.ImportJobPending,
			CreatedAt:          time.Now(),
			IsFirstImport:      len(existingListings) == 0,
		}
		if err := uc.importJobRepo.Create(ctx, &job); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ImportJobCreated, job.ID, eventrecord.Data{
			"connection_id": job.ConnectionID, "default_warehouse_id": job.DefaultWarehouseID,
			"default_currency": job.DefaultCurrency, "is_first_import": job.IsFirstImport,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createImportJob: %w", err)
	}

	// Best-effort: a notification failure shouldn't fail job creation — the
	// job is the source of truth (cmd/scheduler will still pick it up and
	// run it) and the "started" notification is a courtesy, not required
	// for correctness.
	if _, err := uc.notifier.Execute(ctx, appnotifications.CreateNotificationInput{
		Type:    notificationsDomain.TypeImportStarted,
		Title:   "Import started",
		Message: "Importing WooCommerce products…",
	}); err != nil {
		log.Printf("createImportJob: notify started: %v", err)
	}

	return &job, nil
}
