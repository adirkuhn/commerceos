package integrations

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
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
	// BolEconomicOperatorID and BolDeliveryCode are optional, same
	// nil-or-empty-clears semantics as DefaultWarehouseID. Ignored for
	// non-Bol connections (nothing reads them). BolDeliveryCode is
	// validated against ValidBolDeliveryCodes when non-empty.
	BolEconomicOperatorID *string
	BolDeliveryCode       *string
	// BolShippingCostMinorUnits is what one parcel costs (>= 0); nil keeps
	// the current value.
	BolShippingCostMinorUnits *int64
}

// UpdateConnectionSettingsUseCase persists connection-level defaults —
// today just DefaultCurrency, set once here rather than retyped on every
// manual import or scheduled products/import run.
type UpdateConnectionSettingsUseCase struct {
	db       *gorm.DB
	connRepo integrationsDomain.ConnectionRepository
	outbox   eventrecord.Enqueuer
}

func NewUpdateConnectionSettingsUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, outbox eventrecord.Enqueuer) *UpdateConnectionSettingsUseCase {
	return &UpdateConnectionSettingsUseCase{db: db, connRepo: connRepo, outbox: outbox}
}

func (uc *UpdateConnectionSettingsUseCase) Execute(ctx context.Context, in UpdateConnectionSettingsInput) (*integrationsDomain.Connection, error) {
	if in.BolDeliveryCode != nil {
		code := strings.TrimSpace(*in.BolDeliveryCode)
		if code != "" && !integrationsDomain.ValidBolDeliveryCodes[code] {
			return nil, fmt.Errorf("updateConnectionSettings: %w: %q", integrationsDomain.ErrInvalidBolDeliveryCode, code)
		}
	}

	if in.BolShippingCostMinorUnits != nil && *in.BolShippingCostMinorUnits < 0 {
		return nil, fmt.Errorf("updateConnectionSettings: %w", integrationsDomain.ErrInvalidBolShippingCost)
	}

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
		before := *existing
		existing.DefaultCurrency = strings.ToUpper(strings.TrimSpace(in.DefaultCurrency))
		if in.DefaultWarehouseID != nil && strings.TrimSpace(*in.DefaultWarehouseID) != "" {
			warehouseID := strings.TrimSpace(*in.DefaultWarehouseID)
			existing.DefaultWarehouseID = &warehouseID
		} else {
			existing.DefaultWarehouseID = nil
		}
		if in.BolEconomicOperatorID != nil && strings.TrimSpace(*in.BolEconomicOperatorID) != "" {
			economicOperatorID := strings.TrimSpace(*in.BolEconomicOperatorID)
			existing.BolEconomicOperatorID = &economicOperatorID
		} else if in.BolEconomicOperatorID != nil {
			existing.BolEconomicOperatorID = nil
		}
		if in.BolDeliveryCode != nil && strings.TrimSpace(*in.BolDeliveryCode) != "" {
			deliveryCode := strings.TrimSpace(*in.BolDeliveryCode)
			existing.BolDeliveryCode = &deliveryCode
		} else if in.BolDeliveryCode != nil {
			existing.BolDeliveryCode = nil
		}
		if in.BolShippingCostMinorUnits != nil {
			existing.BolShippingCostMinorUnits = *in.BolShippingCostMinorUnits
		}
		existing.UpdatedAt = time.Now()
		if err := uc.connRepo.Update(ctx, existing); err != nil {
			return err
		}
		conn = *existing
		return recordConnectionUpdated(ctx, uc.outbox, conn, eventrecord.NewChanges().
			Set("default_currency", before.DefaultCurrency, conn.DefaultCurrency).
			Set("default_warehouse_id", before.DefaultWarehouseID, conn.DefaultWarehouseID).
			Set("bol_economic_operator_id", before.BolEconomicOperatorID, conn.BolEconomicOperatorID).
			Set("bol_delivery_code", before.BolDeliveryCode, conn.BolDeliveryCode).
			Set("bol_shipping_cost_minor_units", before.BolShippingCostMinorUnits, conn.BolShippingCostMinorUnits))
	})
	if err != nil {
		return nil, fmt.Errorf("updateConnectionSettings: %w", err)
	}
	return &conn, nil
}
