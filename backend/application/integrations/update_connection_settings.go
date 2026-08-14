package integrations

import (
	"context"
	"fmt"
	"strings"
	"time"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateConnectionSettingsInput struct {
	ID              string
	DefaultCurrency string
	// DefaultWarehouseID is optional. nil or pointing to "" clears the
	// connection's preferred warehouse.
	DefaultWarehouseID *string
}

// UpdateConnectionSettingsUseCase persists connection-level defaults —
// today just DefaultCurrency, set once here rather than retyped on every
// manual import or scheduled products/import run.
type UpdateConnectionSettingsUseCase struct {
	db       *gorm.DB
	connRepo integrationsDomain.ConnectionRepository
}

func NewUpdateConnectionSettingsUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository) *UpdateConnectionSettingsUseCase {
	return &UpdateConnectionSettingsUseCase{db: db, connRepo: connRepo}
}

func (uc *UpdateConnectionSettingsUseCase) Execute(ctx context.Context, in UpdateConnectionSettingsInput) (*integrationsDomain.Connection, error) {
	var conn integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		// Currency codes are canonically uppercase (ISO 4217) — normalizing
		// here means every reader (the import modal's <select>, the sync
		// schedule row) can exact-match against it without each one having
		// to re-normalize on read.
		existing.DefaultCurrency = strings.ToUpper(strings.TrimSpace(in.DefaultCurrency))
		if in.DefaultWarehouseID != nil && strings.TrimSpace(*in.DefaultWarehouseID) != "" {
			warehouseID := strings.TrimSpace(*in.DefaultWarehouseID)
			existing.DefaultWarehouseID = &warehouseID
		} else {
			existing.DefaultWarehouseID = nil
		}
		existing.UpdatedAt = time.Now()
		if err := uc.connRepo.Update(ctx, existing); err != nil {
			return err
		}
		conn = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateConnectionSettings: %w", err)
	}
	return &conn, nil
}
