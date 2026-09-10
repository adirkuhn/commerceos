import { Fragment } from 'react'
import { Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'

export interface Crumb {
  label: string
  href?: string
}

export function Breadcrumbs({ items }: { items: Crumb[] }) {
  return (
    <nav aria-label="Breadcrumb" className="text-sm text-muted-foreground">
      <ol className="flex flex-wrap items-center gap-1.5">
        <li><Link to="/" className="hover:text-foreground">Home</Link></li>
        {items.map((item, i) => (
          <Fragment key={item.label}>
            <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
            <li aria-current={i === items.length - 1 ? 'page' : undefined}>
              {item.href && i !== items.length - 1
                ? <Link to={item.href} className="hover:text-foreground">{item.label}</Link>
                : <span className="text-foreground">{item.label}</span>}
            </li>
          </Fragment>
        ))}
      </ol>
    </nav>
  )
}
