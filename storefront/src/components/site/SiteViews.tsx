import { useRenderEnv } from '../../runtime/RenderContext'
import { useCart } from '../../runtime/page/CartContext'
import { useCustomer } from '../../runtime/page/CustomerContext'
import { useStoreInfo } from '../../runtime/page/StoreContext'
import { storeHref } from '../../runtime/page/links'
import type { RendererProps } from '../types'
import type { AccountLinkProps, CartLinkProps, NavigationMenuProps, SearchBoxProps, SiteLogoProps } from './site'

/** Editor canvases never navigate away. */
const inert = (mode: string) => (mode === 'editor' ? { tabIndex: -1, onClick: (e: { preventDefault(): void }) => e.preventDefault() } : {})

export function SiteLogoView({ node, props, className }: RendererProps<SiteLogoProps>) {
  const env = useRenderEnv()
  const store = useStoreInfo()
  const logoID = store?.settings.brand.logo_asset_id
  const logo = logoID ? (store?.assets[logoID] ?? env.assets[logoID]) : undefined
  const name = store?.name || 'Store'
  const size = ['sm', 'md', 'lg'].includes(props.size) ? props.size : 'md'
  return (
    <a href={env.storePath('')} className={`cos-logo cos-logo-${size} ${className}`} data-node={node.id} {...inert(env.mode)}>
      {logo ? <img src={logo.url} alt={name} /> : <span>{name}</span>}
    </a>
  )
}

export function NavigationMenuView({ node, props, className }: RendererProps<NavigationMenuProps>) {
  const env = useRenderEnv()
  const items = (props.items ?? []).slice(0, 12)
  return (
    <nav className={`cos-nav cos-nav-${props.orientation === 'vertical' ? 'vertical' : 'horizontal'} ${className}`} data-node={node.id} aria-label="Menu">
      <ul>
        {items.map((item, i) => {
          const href = storeHref(item.link, env.storePath)
          return <li key={i}>{href ? <a href={href} {...inert(env.mode)}>{item.label}</a> : <span>{item.label}</span>}</li>
        })}
      </ul>
    </nav>
  )
}

export function CartLinkView({ node, props, className }: RendererProps<CartLinkProps>) {
  const env = useRenderEnv()
  const cart = useCart()
  const count = cart?.cart?.items.reduce((n, i) => n + i.quantity, 0) ?? 0
  return (
    <a href={env.storePath('cart')} className={`cos-cart-link ${className}`} data-node={node.id} {...inert(env.mode)}>
      {props.label || 'Cart'}
      {props.show_count !== false && count > 0 && <span className="cos-count" aria-label={`${count} items`}>{count}</span>}
    </a>
  )
}

export function SearchBoxView({ node, props, className }: RendererProps<SearchBoxProps>) {
  const env = useRenderEnv()
  return (
    <form role="search" action={env.storePath('search')} method="get" className={`cos-search ${className}`} data-node={node.id}
      onSubmit={e => { if (env.mode === 'editor') e.preventDefault() }}>
      <input type="search" name="q" aria-label="Search" placeholder={props.placeholder || 'Search'} />
      <button type="submit">Search</button>
    </form>
  )
}

export function AccountLinkView({ node, props, className }: RendererProps<AccountLinkProps>) {
  const env = useRenderEnv()
  const customer = useCustomer()
  const signedIn = !!customer?.account
  return (
    <a href={env.storePath(signedIn ? 'account' : 'login')} className={`cos-account-link ${className}`} data-node={node.id} {...inert(env.mode)}>
      {signedIn ? props.account_label || 'Account' : props.login_label || 'Sign in'}
    </a>
  )
}
