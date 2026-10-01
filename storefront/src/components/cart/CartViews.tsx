import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useShopFormat } from '../../i18n/format'
import { useRenderEnv } from '../../runtime/RenderContext'
import { useCart } from '../../runtime/page/CartContext'
import type { RendererProps } from '../types'
import type { CartItemsProps, CartSummaryProps, CheckoutButtonProps } from './cart'

export function CartItemsView({ node, props, className }: RendererProps<CartItemsProps>) {
  const cart = useCart()
  const env = useRenderEnv()
  const { t } = useTranslation()
  const { price } = useShopFormat()
  if (env.mode === 'editor' && !cart?.cart?.items.length) {
    return <div className={`cos-placeholder ${className}`} data-node={node.id}>{t('hint.cart')}</div>
  }
  if (!cart?.cart) return <div className={`cos-cart-items ${className}`} data-node={node.id} aria-busy="true"><p className="cos-muted">{t('hint.loading')}</p></div>
  if (!cart.cart.items.length) {
    return <p className={`cos-muted ${className}`} data-node={node.id}>{props.empty_message || t('cart.empty')} <a href={env.storePath('')}>{t('common.continueShopping')}</a></p>
  }
  return (
    <ul className={`cos-cart-items ${className}`} data-node={node.id} aria-label={t('cart.items')}>
      {cart.cart.items.map(item => (
        <li key={item.sku_code} className="cos-cart-line">
          <span className="cos-cart-name">{item.product_name}</span>
          <span className="cos-quantity" role="group" aria-label={t('cart.quantityOf', { name: item.product_name })}>
            <button type="button" aria-label={t('product.fewer')} disabled={cart.busy} onClick={() => void cart.setQuantity(item.sku_code, item.quantity - 1)}>−</button>
            <span>{item.quantity}</span>
            <button type="button" aria-label={t('product.more')} disabled={cart.busy} onClick={() => void cart.setQuantity(item.sku_code, item.quantity + 1)}>+</button>
          </span>
          <span>{price(item.price_minor_units * item.quantity, item.currency_code)}</span>
          <button type="button" className="cos-link" disabled={cart.busy} onClick={() => void cart.setQuantity(item.sku_code, 0)}>{t('common.remove')}</button>
        </li>
      ))}
      {cart.error && <li role="alert" className="cos-error">{cart.error}</li>}
    </ul>
  )
}

export function CartSummaryView({ node, props, className }: RendererProps<CartSummaryProps>) {
  const cart = useCart()
  const { t } = useTranslation()
  const { price } = useShopFormat()
  const [code, setCode] = useState('')
  const c = cart?.cart
  const currency = c?.items[0]?.currency_code
  const total = c ? c.subtotal_minor_units - c.discount_minor_units : 0
  return (
    <div className={`cos-cart-summary ${className}`} data-node={node.id}>
      <dl>
        <div><dt>{t('cart.subtotal')}</dt><dd>{c && currency ? price(c.subtotal_minor_units, currency) : '—'}</dd></div>
        {!!c?.discount_minor_units && currency && <div><dt>{t('cart.discount')}</dt><dd>−{price(c.discount_minor_units, currency)}</dd></div>}
        <div className="cos-total"><dt>{t('common.total')}</dt><dd>{c && currency ? price(total, currency) : '—'}</dd></div>
      </dl>
      {props.show_coupon !== false && c && c.items.length > 0 && (
        <form className="cos-coupon" onSubmit={e => { e.preventDefault(); void cart!.applyCoupon(code.trim()) }}>
          <label htmlFor={`${node.id}-coupon`}>{t('cart.couponCode')}</label>
          <input id={`${node.id}-coupon`} value={code} onChange={e => setCode(e.target.value)} placeholder={c.coupon_code ?? ''} />
          <button type="submit" disabled={cart!.busy}>{t('cart.apply')}</button>
        </form>
      )}
    </div>
  )
}

export function CheckoutButtonView({ node, props, className }: RendererProps<CheckoutButtonProps>) {
  const cart = useCart()
  const env = useRenderEnv()
  const { t } = useTranslation()
  const empty = !cart?.cart?.items.length
  const live = env.mode === 'live' && !empty
  return (
    <a className={`cos-button cos-button-primary ${className}`} data-node={node.id} href={live ? env.storePath('checkout') : undefined}
      aria-disabled={live ? undefined : true} role={live ? undefined : 'link'}>
      {props.label || t('cart.checkout')}
    </a>
  )
}
