import { Link, NavLink } from 'react-router-dom'
import { Heart, Menu, ShoppingBag, User, Scale } from 'lucide-react'
import { Container } from './Container'
import { Button } from '@/components/ui/Button'
import { SearchBar } from '@/components/ecommerce/SearchBar'
import { SettingsMenu } from './SettingsMenu'
import { useCart } from '@/features/cart/hooks'
import { useCartStore } from '@/stores/cartStore'
import { useUIStore } from '@/stores/uiStore'
import { useWishlistStore } from '@/stores/wishlistStore'
import { useCompareStore } from '@/stores/compareStore'
import { useCategories } from '@/features/categories/hooks'
import { cn } from '@/lib/utils'
import { pluginRegistry } from '@/plugins'

export function Header() {
  const { data: cart } = useCart()
  const cartCount = cart?.items.reduce((sum, i) => sum + i.quantity, 0) ?? 0
  const wishlistCount = useWishlistStore(s => s.productIds.length)
  const compareCount = useCompareStore(s => s.productIds.length)
  const openDrawer = useCartStore(s => s.openDrawer)
  const openMobileNav = useUIStore(s => s.openMobileNav)
  const { data: categories = [] } = useCategories()
  const pluginNavItems = pluginRegistry.navigationItems('header')

  return (
    <header className="sticky top-0 z-40 border-b border-border bg-background/95 backdrop-blur">
      <Container className="flex h-16 items-center gap-4">
        <button className="lg:hidden" onClick={openMobileNav} aria-label="Open menu">
          <Menu className="h-5 w-5" />
        </button>
        <Link to="/" className="shrink-0 text-lg font-bold tracking-tight">Aurora</Link>

        <nav className="hidden items-center gap-5 lg:flex" aria-label="Primary">
          {categories.slice(0, 6).map(c => (
            <NavLink
              key={c.id} to={`/category/${c.slug}`}
              className={({ isActive }) => cn('text-sm font-medium text-muted-foreground transition-colors hover:text-foreground', isActive && 'text-foreground')}
            >
              {c.name}
            </NavLink>
          ))}
          {pluginNavItems.map(item => (
            <a key={item.href} href={item.href} className="text-sm font-medium text-muted-foreground hover:text-foreground">{item.label}</a>
          ))}
        </nav>

        <div className="ml-auto hidden max-w-sm flex-1 sm:block">
          <SearchBar />
        </div>

        <div className="ml-auto flex items-center gap-1 sm:ml-0">
          <SettingsMenu />
          <Button asChild variant="ghost" size="icon" aria-label={`Compare (${compareCount} products)`}>
            <Link to="/compare" className="relative">
              <Scale className="h-5 w-5" />
              {compareCount > 0 && <CountBadge count={compareCount} />}
            </Link>
          </Button>
          <Button asChild variant="ghost" size="icon" aria-label={`Wishlist (${wishlistCount} items)`}>
            <Link to="/wishlist" className="relative">
              <Heart className="h-5 w-5" />
              {wishlistCount > 0 && <CountBadge count={wishlistCount} />}
            </Link>
          </Button>
          <Button asChild variant="ghost" size="icon" aria-label="Account">
            <Link to="/account"><User className="h-5 w-5" /></Link>
          </Button>
          <Button variant="ghost" size="icon" aria-label={`Cart (${cartCount} items)`} onClick={openDrawer} className="relative">
            <ShoppingBag className="h-5 w-5" />
            {cartCount > 0 && <CountBadge count={cartCount} />}
          </Button>
        </div>
      </Container>
      <div className="border-t border-border p-3 sm:hidden">
        <SearchBar />
      </div>
    </header>
  )
}

function CountBadge({ count }: { count: number }) {
  return (
    <span className="absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-medium text-primary-foreground">
      {count > 99 ? '99+' : count}
    </span>
  )
}
