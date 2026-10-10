package tenant_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestIDFromContext_empty(t *testing.T) {
	assert.Equal(t, "", tenant.IDFromContext(context.Background()))
}

func TestIDFromContext_roundtrip(t *testing.T) {
	ctx := tenant.WithTenantID(context.Background(), "tenant-abc")
	assert.Equal(t, "tenant-abc", tenant.IDFromContext(ctx))
}

func TestIsSuperAdmin_false_by_default(t *testing.T) {
	assert.False(t, tenant.IsSuperAdmin(context.Background()))
}

func TestIsSuperAdmin_true_after_set(t *testing.T) {
	ctx := tenant.WithSuperAdmin(context.Background())
	assert.True(t, tenant.IsSuperAdmin(ctx))
}

func TestWithSuperAdmin_does_not_overwrite_tenant(t *testing.T) {
	ctx := tenant.WithTenantID(context.Background(), "tenant-xyz")
	ctx = tenant.WithSuperAdmin(ctx)
	assert.Equal(t, "tenant-xyz", tenant.IDFromContext(ctx))
	assert.True(t, tenant.IsSuperAdmin(ctx))
}
func TestAsTenant_scopesToTheTenantAndDropsSuperAdmin(t *testing.T) {
	admin := tenant.WithSuperAdmin(context.Background())

	ctx := tenant.AsTenant(admin, "t-1")

	assert.Equal(t, "t-1", tenant.IDFromContext(ctx))
	assert.False(t, tenant.IsSuperAdmin(ctx))
}
