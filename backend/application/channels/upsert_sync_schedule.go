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
	channelsDomain.SyncEntityProducts: true,
	channelsDomain.SyncEntityStock:    true,
	channelsDomain.SyncEntityOrders:   true,
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

func (uc *UpsertSyncScheduleUseCase) Execute(ctx context.Context, in UpsertSyncScheduleInput) (*channelsDomain.SyncSchedule, error) {
	switch in.EntityType {
	case channelsDomain.SyncEntityProducts, channelsDomain.SyncEntityStock,
		channelsDomain.SyncEntityOrders, channelsDomain.SyncEntityCategories:
	default:
		return nil, fmt.Errorf("upsertSyncSchedule: %w", channelsDomain.ErrInvalidSyncEntityType)
	}
	switch in.Direction {
	case channelsDomain.SyncDirectionImport, channelsDomain.SyncDirectionExport:
	default:
		return nil, fmt.Errorf("upsertSyncSchedule: %w", channelsDomain.ErrInvalidSyncDirection)
	}
	if in.Enabled && !runnableSyncEntityTypes[in.EntityType] {
		return nil, fmt.Errorf("upsertSyncSchedule: %w", channelsDomain.ErrCannotEnableUnrunnableEntity)
	}
	// Orders only imports -- this codebase never pushes a CommerceOS-created
	// order back to Woo (see cmd/scheduler/dispatch.go's buildRunners doc).
	if in.Enabled && in.EntityType == channelsDomain.SyncEntityOrders && in.Direction == channelsDomain.SyncDirectionExport {
		return nil, fmt.Errorf("upsertSyncSchedule: %w", channelsDomain.ErrCannotEnableUnrunnableEntity)
	}
	if in.IntervalMinutes <= 0 {
		return nil, fmt.Errorf("upsertSyncSchedule: %w", channelsDomain.ErrInvalidSyncInterval)
	}
	hasWarehouse := in.DefaultWarehouseID != nil && *in.DefaultWarehouseID != ""
	hasCurrency := in.DefaultCurrency != nil && *in.DefaultCurrency != ""
	if in.Enabled {
		switch {
		case in.EntityType == channelsDomain.SyncEntityProducts && in.Direction == channelsDomain.SyncDirectionImport:
			if !hasWarehouse || !hasCurrency {
				return nil, fmt.Errorf("upsertSyncSchedule: %w", channelsDomain.ErrDefaultWarehouseAndCurrencyRequired)
			}
		case in.EntityType == channelsDomain.SyncEntityStock && in.Direction == channelsDomain.SyncDirectionImport,
			in.EntityType == channelsDomain.SyncEntityOrders && in.Direction == channelsDomain.SyncDirectionImport:
			if !hasWarehouse {
				return nil, fmt.Errorf("upsertSyncSchedule: %w", channelsDomain.ErrDefaultWarehouseRequired)
			}
		}
	}

	var schedule channelsDomain.SyncSchedule
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		var nextRunAt *time.Time
		if in.Enabled {
			t := now.Add(time.Duration(in.IntervalMinutes) * time.Minute)
			nextRunAt = &t
		}
		schedule = channelsDomain.SyncSchedule{
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
		return uc.repo.Upsert(ctx, &schedule)
	})
	if err != nil {
		return nil, fmt.Errorf("upsertSyncSchedule: %w", err)
	}
	return &schedule, nil
}
