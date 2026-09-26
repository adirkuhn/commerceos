package tenant

import "context"

// Role is the caller's lm2 JWT role (CLAUDE.md "Auth & RBAC"), propagated
// into context by middleware.JWT so use cases can check it.
type Role string

const (
	RoleSuperAdmin    Role = "super_admin"
	RoleCustomerOwner Role = "customer_owner"
	RoleCustomerAdmin Role = "customer_admin"
	RoleCustomerUser  Role = "customer_user"
)

const roleKey contextKey = "role"

func WithRole(ctx context.Context, role Role) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

func RoleFromContext(ctx context.Context) Role {
	v, _ := ctx.Value(roleKey).(Role)
	return v
}

// IsTenantAdmin: owner or admin of the tenant, or a super admin -- the
// roles allowed irreversible catalog operations (e.g. merging products).
func IsTenantAdmin(ctx context.Context) bool {
	switch RoleFromContext(ctx) {
	case RoleSuperAdmin, RoleCustomerOwner, RoleCustomerAdmin:
		return true
	}
	return false
}
