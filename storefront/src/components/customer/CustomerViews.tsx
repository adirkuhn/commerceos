import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useStorefrontAPI } from '../../api/ApiContext'
import type { AccountOrder } from '../../api/storefront'
import { Trans, useTranslation } from 'react-i18next'
import { shopperError } from '../../i18n/shopperError'
import { useShopFormat } from '../../i18n/format'
import { orderStatusLabel } from '../../i18n/orderStatus'
import { useRenderEnv } from '../../runtime/RenderContext'
import { useCustomer } from '../../runtime/page/CustomerContext'
import { useProductPage } from '../../runtime/page/ProductContext'
import { useWishlist } from '../../runtime/page/WishlistContext'
import { ProductTile } from '../commerce/ProductTile'
import { EditorHint } from '../EditorHint'
import type { RendererProps } from '../types'
import type { AccountOverviewProps, LoginFormProps, WishlistButtonProps, WishlistItemsProps } from './customer'

export function LoginFormView({ node, props, className }: RendererProps<LoginFormProps>) {
  const env = useRenderEnv()
  const customer = useCustomer()
  const { t, i18n } = useTranslation()
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
      () => { setError(t('account.linkExpired')); setState('error') },
    )
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (env.mode === 'editor' || !customer) return
    setState('busy')
    try {
      await customer.requestLink(email)
      setState('sent')
    } catch (err) {
      setError(shopperError(err, i18n, 'common.somethingWrong'))
      setState('error')
    }
  }

  return (
    <section className={`cos-login ${className}`} data-node={node.id}>
      {props.title && <h1 className="cos-heading">{props.title}</h1>}
      {state === 'signing-in' ? <p role="status">{t('account.signingIn')}</p>
        : state === 'sent' ? <p role="status">{t('account.linkSent')}</p>
        : (
          <form onSubmit={e => void submit(e)}>
            <label htmlFor={`${node.id}-email`} className="cos-label">{t('common.email')}</label>
            <input id={`${node.id}-email`} type="email" required autoComplete="email" value={email} onChange={e => setEmail(e.target.value)} />
            <button type="submit" className="cos-button cos-button-primary" disabled={state === 'busy'}>{props.button_label || t('account.emailMeLink')}</button>
            {state === 'error' && <p role="alert" className="cos-error">{error}</p>}
          </form>
        )}
    </section>
  )
}

function Orders({ sessionToken }: { sessionToken: string }) {
  const api = useStorefrontAPI()
  const { date, price } = useShopFormat()
  const { t } = useTranslation()
  const { data, isLoading, isError } = useQuery({ queryKey: ['account-orders', sessionToken], queryFn: () => api.listAccountOrders(sessionToken) })
  if (isLoading) return <p className="cos-muted">{t('account.loadingOrders')}</p>
  if (isError) return <p role="alert" className="cos-error">{t('account.ordersFailed')}</p>
  if (!data?.length) return <p className="cos-muted">{t('account.noOrders')}</p>
  return (
    <table className="cos-orders">
      <thead><tr><th>{t('common.order')}</th><th>{t('common.date')}</th><th>{t('common.status')}</th><th>{t('common.total')}</th></tr></thead>
      <tbody>
        {data.map((o: AccountOrder) => (
          <tr key={o.order_number}>
            <td>{o.order_number}</td><td>{date(o.placed_at)}</td><td>{orderStatusLabel(t, o.status)}</td><td>{price(o.total_minor_units, o.currency_code)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

export function AccountOverviewView({ node, props, className }: RendererProps<AccountOverviewProps>) {
  const env = useRenderEnv()
  const customer = useCustomer()
  const { t } = useTranslation()
  if (!customer || env.mode === 'editor') {
    return <EditorHint nodeId={node.id} className={className} hint="account" />
  }
  if (customer.account === undefined) return <div className={className} data-node={node.id} aria-busy="true" />
  if (!customer.account) {
    return <p className={className} data-node={node.id}><Trans i18nKey="account.signInFirst" components={{ link: <a href={env.storePath('login')} /> }} /></p>
  }
  const a = customer.account
  return (
    <section className={`cos-account ${className}`} data-node={node.id}>
      <p><strong>{[a.first_name, a.last_name].filter(Boolean).join(' ') || a.email}</strong><br />{a.email}</p>
      {props.show_orders !== false && customer.sessionToken && <Orders sessionToken={customer.sessionToken} />}
      <button type="button" className="cos-link" onClick={() => void customer.signOut().then(() => env.navigate(env.storePath('')))}>{t('account.signOut')}</button>
    </section>
  )
}

export function WishlistItemsView({ node, props, className }: RendererProps<WishlistItemsProps>) {
  const api = useStorefrontAPI()
  const env = useRenderEnv()
  const wishlist = useWishlist()
  const { t } = useTranslation()
  const ids = wishlist?.productIds
  const { data } = useQuery({
    queryKey: ['wishlist-products', ids], enabled: !!ids?.length,
    queryFn: () => api.listProducts({ productIds: ids, perPage: Math.max(ids?.length ?? 1, 1) }),
  })
  if (!wishlist || env.mode === 'editor') return <EditorHint nodeId={node.id} className={className} hint="wishlist" />
  if (ids === undefined) return <div className={className} data-node={node.id} aria-busy="true" />
  if (!ids.length) return <p className={`cos-muted ${className}`} data-node={node.id}>{props.empty_message || t('wishlist.empty')}</p>
  return (
    <ul className={`cos-product-grid ${className}`} data-node={node.id} aria-label={t('wishlist.title')}>
      {(data?.items ?? []).map(p => (
        <li key={p.id}>
          <ProductTile product={p} showPrice />
          <button type="button" className="cos-link" disabled={wishlist.busy} onClick={() => void wishlist.toggle(p.id)}>{t('common.remove')}</button>
        </li>
      ))}
    </ul>
  )
}

export function WishlistButtonView({ node, props, className }: RendererProps<WishlistButtonProps>) {
  const page = useProductPage()
  const wishlist = useWishlist()
  const { t } = useTranslation()
  const env = useRenderEnv()
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="wishlistButton" />
  const saved = !!wishlist?.productIds?.includes(page.product.id)
  return (
    <button type="button" aria-pressed={saved} className={`cos-wishlist-button ${className}`} data-node={node.id}
      disabled={!wishlist || wishlist.busy || env.mode === 'editor'} onClick={() => void wishlist?.toggle(page.product.id)}>
      {saved ? props.remove_label || t('product.saved') : props.add_label || t('product.saveToWishlist')}
    </button>
  )
}
