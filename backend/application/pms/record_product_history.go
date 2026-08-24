package pms

import (
	"context"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// recordProductHistoryFieldIfChanged writes one ProductHistory row when
// oldValue and newValue differ, no-op otherwise -- called from inside the
// same tenant.WithTenant transaction as the entity write itself, so the
// audit trail is atomic with the change it records.
func recordProductHistoryFieldIfChanged(
	ctx context.Context,
	repo pmsDomain.ProductHistoryRepository,
	entityType pmsDomain.ProductHistoryEntityType,
	entityID string,
	field pmsDomain.ProductHistoryField,
	oldValue, newValue string,
) error {
	if oldValue == newValue {
		return nil
	}
	return repo.Create(ctx, &pmsDomain.ProductHistory{
		TenantID: tenant.IDFromContext(ctx), EntityType: entityType, EntityID: entityID,
		Field: field, OldValue: oldValue, NewValue: newValue, ChangedAt: time.Now(),
	})
}
