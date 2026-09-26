// The tenant admin roles (owner, admin, super admin) -- mirrors the
// backend's tenant.IsTenantAdmin. Only decides what the UI offers; the
// backend enforces.
const TENANT_ADMIN_ROLES = new Set(['super_admin', 'customer_owner', 'customer_admin'])

export function isTenantAdmin(role: string | undefined): boolean {
  return role !== undefined && TENANT_ADMIN_ROLES.has(role)
}
