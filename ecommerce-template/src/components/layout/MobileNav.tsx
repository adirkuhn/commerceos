import { Link } from 'react-router-dom'
import { Drawer, DrawerContent, DrawerTitle } from '@/components/ui/Drawer'
import { useUIStore } from '@/stores/uiStore'
import { useCategories } from '@/features/categories/hooks'

const STATIC_LINKS = [
  { label: 'Search', href: '/search' },
  { label: 'Wishlist', href: '/wishlist' },
  { label: 'Compare', href: '/compare' },
  { label: 'Blog', href: '/blog' },
  { label: 'Account', href: '/account' },
]

export function MobileNav() {
  const isOpen = useUIStore(s => s.isMobileNavOpen)
  const close = useUIStore(s => s.closeMobileNav)
  const { data: categories = [] } = useCategories()

  return (
    <Drawer open={isOpen} onOpenChange={(open) => !open && close()}>
      <DrawerContent side="left" aria-describedby={undefined}>
        <div className="border-b border-border p-4">
          <DrawerTitle className="text-base font-semibold">Menu</DrawerTitle>
        </div>
        <nav className="flex-1 overflow-y-auto p-4" aria-label="Mobile">
          <p className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Shop</p>
          <ul className="mb-6 space-y-1">
            {categories.map(c => (
              <li key={c.id}><Link to={`/category/${c.slug}`} onClick={close} className="block rounded-md px-2 py-2 text-sm hover:bg-secondary">{c.name}</Link></li>
            ))}
          </ul>
          <ul className="space-y-1">
            {STATIC_LINKS.map(link => (
              <li key={link.href}><Link to={link.href} onClick={close} className="block rounded-md px-2 py-2 text-sm hover:bg-secondary">{link.label}</Link></li>
            ))}
          </ul>
        </nav>
      </DrawerContent>
    </Drawer>
  )
}
