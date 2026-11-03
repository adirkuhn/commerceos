package channels

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

var runnableSyncEntityTypes = map[channelsDomain.SyncEntityType]bool{
	channelsDomain.SyncEntityOrders:               true,
	channelsDomain.SyncEntityOrdersReconciliation: true,
	channelsDomain.SyncEntityBolOrders:            true,
	channelsDomain.SyncEntityBolProducts:          true,
	channelsDomain.SyncEntityBolStock:             true,
}

type UpsertSyncScheduleInput struct {
	ConnectionID    string
	EntityType      channelsDomain.SyncEntityType
	Direction       channelsDomain.SyncDirection
	Enabled         bool
	IntervalMinutes int
	// DefaultWarehouseID is required when Enabled=true and Direction=import
	// for EntityType orders/orders_reconciliation/bol_orders — see
	// SyncSchedule's doc comment. Ignored for every other combination.
	DefaultWarehouseID *string
	DefaultCurrency    *string
}

type UpsertSyncScheduleUseCase struct {
	db     *gorm.DB
	repo   channelsDomain.SyncScheduleRepository
	outbox eventrecord.Enqueuer
}

func NewUpsertSyncScheduleUseCase(db *gorm.DB, repo channelsDomain.SyncScheduleRepository, outbox eventrecord.Enqueuer) *UpsertSyncScheduleUseCase {
	return &UpsertSyncScheduleUseCase{db: db, repo: repo, outbox: outbox}
}

// validateUpsertSyncScheduleInput is shared by UpsertSyncScheduleUseCase
// (single-row) and UpsertSyncSchedulesUseCase (the batch the settings page
// PUT actually calls) -- kept as one function so the two never drift.
func validateUpsertSyncScheduleInput(in UpsertSyncScheduleInput) error {
	switch in.EntityType {
	case channelsDomain.SyncEntityOrders, channelsDomain.SyncEntityCategories,
		channelsDomain.SyncEntityOrdersReconciliation,
		channelsDomain.SyncEntityBolOrders, channelsDomain.SyncEntityBolProducts, channelsDomain.SyncEntityBolStock:
	default:
		return channelsDomain.ErrInvalidSyncEntityType
	}
	switch in.Direction {
	case channelsDomain.SyncDirectionImport, channelsDomain.SyncDirectionExport:
	default:
		return channelsDomain.ErrInvalidSyncDirection
	}
	if in.Enabled && !runnableSyncEntityTypes[in.EntityType] {
		return channelsDomain.ErrCannotEnableUnrunnableEntity
	}
	// Orders (both the incremental schedule and its reconciliation
	// counterpart) only import -- this codebase never pushes a
	// CommerceOS-created order back to Woo (see cmd/scheduler/dispatch.go's
	// buildRunners doc). Same for Bol orders.
	if in.Enabled && in.Direction == channelsDomain.SyncDirectionExport &&
		(in.EntityType == channelsDomain.SyncEntityOrders || in.EntityType == channelsDomain.SyncEntityOrdersReconciliation ||
			in.EntityType == channelsDomain.SyncEntityBolOrders) {
		return channelsDomain.ErrCannotEnableUnrunnableEntity
	}
	// bol_products/bol_stock are export-only -- Bol never hands work back
	// to CommerceOS for these (no product/stock import from Bol).
	if in.Enabled && in.Direction == channelsDomain.SyncDirectionImport &&
		(in.EntityType == channelsDomain.SyncEntityBolProducts || in.EntityType == channelsDomain.SyncEntityBolStock) {
		return channelsDomain.ErrCannotEnableUnrunnableEntity
	}
	if in.IntervalMinutes <= 0 {
		return channelsDomain.ErrInvalidSyncInterval
	}
	hasWarehouse := in.DefaultWarehouseID != nil && *in.DefaultWarehouseID != ""
	if in.Enabled {
		switch {
		case in.Direction == channelsDomain.SyncDirectionImport &&
			(in.EntityType == channelsDomain.SyncEntityOrders || in.EntityType == channelsDomain.SyncEntityOrdersReconciliation):
			if !hasWarehouse {
				return channelsDomain.ErrDefaultWarehouseRequired
			}
		case in.EntityType == channelsDomain.SyncEntityBolOrders && in.Direction == channelsDomain.SyncDirectionImport:
			// No DefaultCurrency requirement -- Bol prices are already
			// assumed EUR throughout this codebase's Bol code (see
			// BolOfferInput/BolProductOverride, neither has a currency
			// field), unlike Woo's products/import.
			if !hasWarehouse {
				return channelsDomain.ErrDefaultWarehouseRequired
			}
		}
	}
	return nil
}

// buildSyncSchedule maps a validated input into the persisted-row shape --
// shared by both use cases for the same drift-avoidance reason as
// validateUpsertSyncScheduleInput.
func buildSyncSchedule(ctx context.Context, in UpsertSyncScheduleInput, now time.Time) channelsDomain.SyncSchedule {
	var nextRunAt *time.Time
	if in.Enabled {
		t := now.Add(time.Duration(in.IntervalMinutes) * time.Minute)
		nextRunAt = &t
	}
	return channelsDomain.SyncSchedule{
		TenantID:           tenant.IDFromContext(ctx),
		ConnectionID:       in.ConnectionID,
		EntityType:         in.EntityType,
		Direction:          in.Direction,
		Enabled:            in.Enabled,
		IntervalMinutes:    in.IntervalMinutes,
		DefaultWarehouseID: in.DefaultWarehouseID,
		DefaultCurrency:    in.DefaultCurrency,
		NextRunAt:          nextRunAt,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func (uc *UpsertSyncScheduleUseCase) Execute(ctx context.Context, in UpsertSyncScheduleInput) (*channelsDomain.SyncSchedule, error) {
	if err := validateUpsertSyncScheduleInput(in); err != nil {
		return nil, fmt.Errorf("upsertSyncSchedule: %w", err)
	}

	var schedule channelsDomain.SyncSchedule
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		schedule = buildSyncSchedule(ctx, in, time.Now())
		if err := uc.repo.Upsert(ctx, &schedule); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.SyncScheduleSaved, schedule.ID, syncScheduleData(schedule))
	})
	if err != nil {
		return nil, fmt.Errorf("upsertSyncSchedule: %w", err)
	}
	return &schedule, nil
}
