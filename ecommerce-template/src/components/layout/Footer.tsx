import { Link } from 'react-router-dom'
import { Truck, RotateCcw, ShieldCheck, Headphones } from 'lucide-react'
import { Container } from './Container'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import { pluginRegistry } from '@/plugins'

const BENEFITS = [
  { icon: Truck, title: 'Free shipping over $100', description: 'On every order, no code needed' },
  { icon: RotateCcw, title: '60-day returns', description: 'Unworn items, full refund' },
  { icon: ShieldCheck, title: 'Secure checkout', description: 'Your data is always protected' },
  { icon: Headphones, title: 'Real support', description: 'From people who use the gear' },
]

const LINK_COLUMNS = [
  { title: 'Shop', links: [{ label: 'New arrivals', href: '/search?filter=new_arrivals' }, { label: 'Best sellers', href: '/search?filter=best_sellers' }, { label: 'Sale', href: '/search?onSale=true' }] },
  { title: 'Help', links: [{ label: 'Shipping & returns', href: '/page/shipping-returns' }, { label: 'FAQ', href: '/page/about' }, { label: 'Track order', href: '/order-lookup' }] },
  { title: 'Company', links: [{ label: 'About Aurora', href: '/page/about' }, { label: 'Journal', href: '/blog' }] },
]

export function Footer() {
  const pluginNavItems = pluginRegistry.navigationItems('footer')

  return (
    <footer className="border-t border-border bg-secondary/30">
      <Container className="py-12">
        <div className="grid grid-cols-2 gap-6 border-b border-border pb-10 sm:grid-cols-4">
          {BENEFITS.map(b => (
            <div key={b.title} className="flex items-start gap-3">
              <b.icon className="h-6 w-6 shrink-0 text-primary" />
              <div>
                <p className="text-sm font-medium">{b.title}</p>
                <p className="text-xs text-muted-foreground">{b.description}</p>
              </div>
            </div>
          ))}
        </div>

        <div className="grid gap-8 py-10 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <h3 className="text-lg font-bold">Aurora</h3>
            <p className="mt-2 max-w-xs text-sm text-muted-foreground">Gear built for people who spend real time outside.</p>
            <form className="mt-4 flex gap-2" onSubmit={(e) => e.preventDefault()}>
              <Input type="email" placeholder="Email address" aria-label="Newsletter email" required />
              <Button type="submit" size="sm">Subscribe</Button>
            </form>
          </div>
          {LINK_COLUMNS.map(col => (
            <div key={col.title}>
              <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{col.title}</p>
              <ul className="mt-3 space-y-2">
                {col.links.map(link => (
                  <li key={link.href}><Link to={link.href} className="text-sm text-muted-foreground hover:text-foreground">{link.label}</Link></li>
                ))}
                {col.title === 'Company' && pluginNavItems.map(item => (
                  <li key={item.href}><a href={item.href} className="text-sm text-muted-foreground hover:text-foreground">{item.label}</a></li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="flex flex-col items-center justify-between gap-3 border-t border-border pt-6 text-xs text-muted-foreground sm:flex-row">
          <p>© {new Date().getFullYear()} Aurora. Demo storefront -- no real payments are processed.</p>
          <div className="flex gap-4">
            <Link to="/page/shipping-returns" className="hover:text-foreground">Privacy</Link>
            <Link to="/page/shipping-returns" className="hover:text-foreground">Terms</Link>
          </div>
        </div>
      </Container>
    </footer>
  )
}
