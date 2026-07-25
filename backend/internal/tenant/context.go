package tenant

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/internal/txctx"
)

type contextKey string

const tenantKey contextKey = "tenant_id"
const superAdminKey contextKey = "is_super_admin"

const bypassRLS = "bypass_rls"

func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

func WithSuperAdmin(ctx context.Context) context.Context {
	return context.WithValue(ctx, superAdminKey, true)
}

func IDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(tenantKey).(string)
	return v
}

func IsSuperAdmin(ctx context.Context) bool {
	v, _ := ctx.Value(superAdminKey).(bool)
	return v
}

// WithTenant opens a transaction, sets RLS, embeds tx in ctx, then calls fn.
// All reads and writes must go through this — it satisfies both the transaction
// rule and the RLS rule simultaneously.
func WithTenant(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	setting := IDFromContext(ctx)
	if setting == "" || IsSuperAdmin(ctx) {
		setting = bypassRLS
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL app.current_tenant = ?", setting).Error; err != nil {
			return fmt.Errorf("set rls tenant: %w", err)
		}
		return fn(txctx.WithTx(ctx, tx))
	})
}
