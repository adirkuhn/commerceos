import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useStorefrontAPI } from '../../api/ApiContext'
import type { AccountOrder } from '../../api/storefront'
import { formatPrice } from '../../core/money'
import { useRenderEnv } from '../../runtime/RenderContext'
import { useCustomer } from '../../runtime/page/CustomerContext'
import { useProductPage } from '../../runtime/page/ProductContext'
import { useWishlist } from '../../runtime/page/WishlistContext'
import { useDateLabel } from '../commerce/dates'
import { ProductTile } from '../commerce/ProductTile'
import { EditorHint } from '../EditorHint'
import type { RendererProps } from '../types'
import type { AccountOverviewProps, LoginFormProps, WishlistButtonProps, WishlistItemsProps } from './customer'

export function LoginFormView({ node, props, className }: RendererProps<LoginFormProps>) {
  const env = useRenderEnv()
  const customer = useCustomer()
  const [email, setEmail] = useState('')
  const [state, setState] = useState<'idle' | 'busy' | 'sent' | 'signing-in' | 'error'>('idle')
  const [error, setError] = useState('')

  // Arriving from the emailed link: /login?token=...
  useEffect(() => {
    if (env.mode !== 'live' || !customer) return
    const token = new URLSearchParams(window.location.search).get('token')
    if (!token) return
    setState('signing-in')
    customer.completeSignIn(token).then(
      () => env.navigate(env.storePath('account'), { replace: true }),
      () => { setError('This sign-in link has expired or was already used. Request a new one below.'); setState('error') },
    )
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (env.mode === 'editor' || !customer) return
    setState('busy')
    try {
      await customer.requestLink(email, window.location.origin + env.storePath('login'))
      setState('sent')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
      setState('error')
    }
  }

  return (
    <section className={`cos-login ${className}`} data-node={node.id}>
      {props.title && <h1 className="cos-heading">{props.title}</h1>}
      {state === 'signing-in' ? <p role="status">Signing you in…</p>
        : state === 'sent' ? <p role="status">Check your inbox: we sent you a sign-in link. It works for 15 minutes.</p>
        : (
          <form onSubmit={e => void submit(e)}>
            <label htmlFor={`${node.id}-email`} className="cos-label">Email</label>
            <input id={`${node.id}-email`} type="email" required autoComplete="email" value={email} onChange={e => setEmail(e.target.value)} />
            <button type="submit" className="cos-button cos-button-primary" disabled={state === 'busy'}>{props.button_label || 'Email me a sign-in link'}</button>
            {state === 'error' && <p role="alert" className="cos-error">{error}</p>}
          </form>
        )}
    </section>
  )
}

function Orders({ sessionToken }: { sessionToken: string }) {
  const api = useStorefrontAPI()
  const date = useDateLabel()
  const { data, isLoading, isError } = useQuery({ queryKey: ['account-orders', sessionToken], queryFn: () => api.listAccountOrders(sessionToken) })
  if (isLoading) return <p className="cos-muted">Loading orders…</p>
  if (isError) return <p role="alert" className="cos-error">Your orders couldn't be loaded.</p>
  if (!data?.length) return <p className="cos-muted">No orders yet.</p>
  return (
    <table className="cos-orders">
      <thead><tr><th>Order</th><th>Date</th><th>Status</th><th>Total</th></tr></thead>
      <tbody>
        {data.map((o: AccountOrder) => (
          <tr key={o.order_number}>
            <td>{o.order_number}</td><td>{date(o.placed_at)}</td><td>{o.status}</td><td>{formatPrice(o.total_minor_units, o.currency_code)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

export function AccountOverviewView({ node, props, className }: RendererProps<AccountOverviewProps>) {
  const env = useRenderEnv()
  const customer = useCustomer()
  if (!customer || env.mode === 'editor') {
    return <EditorHint nodeId={node.id} className={className}>The signed-in shopper's details and orders.</EditorHint>
  }
  if (customer.account === undefined) return <div className={className} data-node={node.id} aria-busy="true" />
  if (!customer.account) {
    return <p className={className} data-node={node.id}>Please <a href={env.storePath('login')}>sign in</a> to see your account.</p>
  }
  const a = customer.account
  return (
    <section className={`cos-account ${className}`} data-node={node.id}>
      <p><strong>{[a.first_name, a.last_name].filter(Boolean).join(' ') || a.email}</strong><br />{a.email}</p>
      {props.show_orders !== false && customer.sessionToken && <Orders sessionToken={customer.sessionToken} />}
      <button type="button" className="cos-link" onClick={() => void customer.signOut().then(() => env.navigate(env.storePath('')))}>Sign out</button>
    </section>
  )
}

export function WishlistItemsView({ node, props, className }: RendererProps<WishlistItemsProps>) {
  const api = useStorefrontAPI()
  const env = useRenderEnv()
  const wishlist = useWishlist()
  const ids = wishlist?.productIds
  const { data } = useQuery({
    queryKey: ['wishlist-products', ids], enabled: !!ids?.length,
    queryFn: () => api.listProducts({ productIds: ids, perPage: Math.max(ids?.length ?? 1, 1) }),
  })
  if (!wishlist || env.mode === 'editor') return <EditorHint nodeId={node.id} className={className}>Products the shopper saved.</EditorHint>
  if (ids === undefined) return <div className={className} data-node={node.id} aria-busy="true" />
  if (!ids.length) return <p className={`cos-muted ${className}`} data-node={node.id}>{props.empty_message || 'Nothing saved yet.'}</p>
  return (
    <ul className={`cos-product-grid ${className}`} data-node={node.id} aria-label="Saved products">
      {(data?.items ?? []).map(p => (
        <li key={p.id}>
          <ProductTile product={p} showPrice />
          <button type="button" className="cos-link" disabled={wishlist.busy} onClick={() => void wishlist.toggle(p.id)}>Remove</button>
        </li>
      ))}
    </ul>
  )
}

export function WishlistButtonView({ node, props, className }: RendererProps<WishlistButtonProps>) {
  const page = useProductPage()
  const wishlist = useWishlist()
  const env = useRenderEnv()
  if (!page) return <EditorHint nodeId={node.id} className={className}>Saves the product this page is about.</EditorHint>
  const saved = !!wishlist?.productIds?.includes(page.product.id)
  return (
    <button type="button" aria-pressed={saved} className={`cos-wishlist-button ${className}`} data-node={node.id}
      disabled={!wishlist || wishlist.busy || env.mode === 'editor'} onClick={() => void wishlist?.toggle(page.product.id)}>
      {saved ? props.remove_label || 'Saved' : props.add_label || 'Save to wishlist'}
    </button>
  )
}
