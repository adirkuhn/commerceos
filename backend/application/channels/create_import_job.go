package channels

import (
	"context"
	"fmt"
	"log"
	"time"

	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
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
	notifier      notifier
}

func NewCreateImportJobUseCase(db *gorm.DB, importJobRepo channelsDomain.ImportJobRepository, notifier notifier) *CreateImportJobUseCase {
	return &CreateImportJobUseCase{db: db, importJobRepo: importJobRepo, notifier: notifier}
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

		job = channelsDomain.ImportJob{
			TenantID:           tenant.IDFromContext(ctx),
			ConnectionID:       in.ConnectionID,
			DefaultWarehouseID: in.DefaultWarehouseID,
			DefaultCurrency:    in.DefaultCurrency,
			Status:             channelsDomain.ImportJobPending,
			CreatedAt:          time.Now(),
		}
		return uc.importJobRepo.Create(ctx, &job)
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
