package channels

import (
	"context"
	"fmt"
	"time"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

var runnableSyncEntityTypes = map[channelsDomain.SyncEntityType]bool{
	channelsDomain.SyncEntityProducts:             true,
	channelsDomain.SyncEntityStock:                true,
	channelsDomain.SyncEntityOrders:               true,
	channelsDomain.SyncEntityOrdersReconciliation: true,
}

type UpsertSyncScheduleInput struct {
	ConnectionID    string
	EntityType      channelsDomain.SyncEntityType
	Direction       channelsDomain.SyncDirection
	Enabled         bool
	IntervalMinutes int
	// DefaultWarehouseID is required when Enabled=true and Direction=import
	// for EntityType products/stock/orders — see SyncSchedule's doc
	// comment. DefaultCurrency is additionally required for products.
	// Both ignored for every other combination.
	DefaultWarehouseID *string
	DefaultCurrency    *string
}

type UpsertSyncScheduleUseCase struct {
	db   *gorm.DB
	repo channelsDomain.SyncScheduleRepository
}

func NewUpsertSyncScheduleUseCase(db *gorm.DB, repo channelsDomain.SyncScheduleRepository) *UpsertSyncScheduleUseCase {
	return &UpsertSyncScheduleUseCase{db: db, repo: repo}
}

// validateUpsertSyncScheduleInput is shared by UpsertSyncScheduleUseCase
// (single-row) and UpsertSyncSchedulesUseCase (the batch the settings page
// PUT actually calls) -- kept as one function so the two never drift.
func validateUpsertSyncScheduleInput(in UpsertSyncScheduleInput) error {
	switch in.EntityType {
	case channelsDomain.SyncEntityProducts, channelsDomain.SyncEntityStock,
		channelsDomain.SyncEntityOrders, channelsDomain.SyncEntityCategories,
		channelsDomain.SyncEntityOrdersReconciliation:
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
	// buildRunners doc).
	if in.Enabled && in.Direction == channelsDomain.SyncDirectionExport &&
		(in.EntityType == channelsDomain.SyncEntityOrders || in.EntityType == channelsDomain.SyncEntityOrdersReconciliation) {
		return channelsDomain.ErrCannotEnableUnrunnableEntity
	}
	if in.IntervalMinutes <= 0 {
		return channelsDomain.ErrInvalidSyncInterval
	}
	hasWarehouse := in.DefaultWarehouseID != nil && *in.DefaultWarehouseID != ""
	hasCurrency := in.DefaultCurrency != nil && *in.DefaultCurrency != ""
	if in.Enabled {
		switch {
		case in.EntityType == channelsDomain.SyncEntityProducts && in.Direction == channelsDomain.SyncDirectionImport:
			if !hasWarehouse || !hasCurrency {
				return channelsDomain.ErrDefaultWarehouseAndCurrencyRequired
			}
		case in.Direction == channelsDomain.SyncDirectionImport &&
			(in.EntityType == channelsDomain.SyncEntityStock || in.EntityType == channelsDomain.SyncEntityOrders ||
				in.EntityType == channelsDomain.SyncEntityOrdersReconciliation):
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
		return uc.repo.Upsert(ctx, &schedule)
	})
	if err != nil {
		return nil, fmt.Errorf("upsertSyncSchedule: %w", err)
	}
	return &schedule, nil
}
