package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// MessageCenterSetupRepo implements mcDomain.SetupRepository.
type MessageCenterSetupRepo struct {
	db *gorm.DB
}

func NewMessageCenterSetupRepo(db *gorm.DB) *MessageCenterSetupRepo {
	return &MessageCenterSetupRepo{db: db}
}

var _ mcDomain.SetupRepository = (*MessageCenterSetupRepo)(nil)

func (r *MessageCenterSetupRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("messageCenterSetupRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Claim inserts the tenant's row; a concurrent claim blocks on the primary
// key until the first commits, then inserts nothing.
func (r *MessageCenterSetupRepo) Claim(ctx context.Context, at time.Time) (bool, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return false, err
	}
	res := tx.Exec(`INSERT INTO message_center_setups (tenant_id, seeded_at) VALUES (?, ?) ON CONFLICT (tenant_id) DO NOTHING`,
		tenant.IDFromContext(ctx), at)
	if res.Error != nil {
		return false, fmt.Errorf("messageCenterSetupRepo.Claim: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
}

// ListUnseededTenants takes the tenants it knows about from their settings
// and Message Center rows.
func (r *MessageCenterSetupRepo) ListUnseededTenants(ctx context.Context) ([]string, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var ids []string
	err = tx.Raw(`
		SELECT tenant_id::text FROM (
			SELECT tenant_id FROM tenant_settings
			UNION SELECT tenant_id FROM message_templates
			UNION SELECT tenant_id FROM automation_rules
		) known
		WHERE tenant_id NOT IN (SELECT tenant_id FROM message_center_setups)
		ORDER BY tenant_id`).Scan(&ids).Error
	if err != nil {
		return nil, fmt.Errorf("messageCenterSetupRepo.ListUnseededTenants: %w", err)
	}
	return ids, nil
}
