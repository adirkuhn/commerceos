import { Link } from 'react-router-dom'

export interface NavMenuLink {
  label: string
  url: string
}

interface Props {
  links: NavMenuLink[]
}

// NavMenuBlock is a merchant-ordered list of label/url links -- not tied
// to any one concept (categories, legal pages, ...), so the same block
// type works for the header page's category browse links and the footer
// page's privacy/terms/returns links. An internal path (starts with "/")
// navigates client-side via react-router's Link; anything else (an
// external URL, e.g. a WhatsApp/social link) is a plain anchor.
export function NavMenuBlock({ links }: Props) {
  return (
    <nav className="flex h-full items-center gap-4 text-sm">
      {links.map((link, i) => (
        link.url.startsWith('/') ? (
          <Link key={i} to={link.url} className="text-brand-500 hover:text-brand-700">{link.label}</Link>
        ) : (
          <a key={i} href={link.url} target="_blank" rel="noreferrer" className="text-brand-500 hover:text-brand-700">{link.label}</a>
        )
      ))}
    </nav>
  )
}
