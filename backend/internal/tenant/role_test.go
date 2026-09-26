package tenant_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestRoleFromContext(t *testing.T) {
	ctx := tenant.WithRole(context.Background(), tenant.RoleCustomerAdmin)
	assert.Equal(t, tenant.RoleCustomerAdmin, tenant.RoleFromContext(ctx))
	assert.Equal(t, tenant.Role(""), tenant.RoleFromContext(context.Background()))
}

func TestIsTenantAdmin(t *testing.T) {
	for role, want := range map[tenant.Role]bool{
		tenant.RoleSuperAdmin: true, tenant.RoleCustomerOwner: true, tenant.RoleCustomerAdmin: true,
		tenant.RoleCustomerUser: false, "": false,
	} {
		assert.Equal(t, want, tenant.IsTenantAdmin(tenant.WithRole(context.Background(), role)), string(role))
	}
}
