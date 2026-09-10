import { NavLink, Outlet, Navigate } from 'react-router-dom'
import { Container } from './Container'
import { useAuthStore } from '@/stores/authStore'
import { cn } from '@/lib/utils'
import { pluginRegistry } from '@/plugins'

const NAV_ITEMS = [
  { label: 'Dashboard', to: '/account' },
  { label: 'Orders', to: '/account/orders' },
  { label: 'Addresses', to: '/account/addresses' },
  { label: 'Wishlist', to: '/account/wishlist' },
  { label: 'Returns', to: '/account/returns' },
  { label: 'Profile', to: '/account/profile' },
]

export function AccountLayout() {
  const isAuthenticated = useAuthStore(s => s.isAuthenticated)
  const accountExtensions = pluginRegistry.all().flatMap(p => p.accountExtensions ?? [])

  if (!isAuthenticated) return <Navigate to="/login?next=/account" replace />

  return (
    <Container className="grid gap-10 py-10 lg:grid-cols-[14rem_1fr]">
      <nav aria-label="Account" className="space-y-1">
        {NAV_ITEMS.map(item => (
          <NavLink
            key={item.to} to={item.to} end={item.to === '/account'}
            className={({ isActive }) => cn('block rounded-md px-3 py-2 text-sm font-medium text-muted-foreground hover:bg-secondary', isActive && 'bg-secondary text-foreground')}
          >
            {item.label}
          </NavLink>
        ))}
        {accountExtensions.map(ext => (
          <NavLink key={ext.id} to={ext.path} className={({ isActive }) => cn('block rounded-md px-3 py-2 text-sm font-medium text-muted-foreground hover:bg-secondary', isActive && 'bg-secondary text-foreground')}>
            {ext.label}
          </NavLink>
        ))}
      </nav>
      <div>
        <Outlet />
      </div>
    </Container>
  )
}
