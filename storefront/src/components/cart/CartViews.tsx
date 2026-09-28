import { useState } from 'react'
import { formatPrice } from '../../core/money'
import { useRenderEnv } from '../../runtime/RenderContext'
import { useCart } from '../../runtime/page/CartContext'
import type { RendererProps } from '../types'
import type { CartItemsProps, CartSummaryProps, CheckoutButtonProps } from './cart'

export function CartItemsView({ node, props, className }: RendererProps<CartItemsProps>) {
  const cart = useCart()
  const env = useRenderEnv()
  if (env.mode === 'editor' && !cart?.cart?.items.length) {
    return <div className={`cos-placeholder ${className}`} data-node={node.id}>The shopper's cart items appear here.</div>
  }
  if (!cart?.cart) return <div className={`cos-cart-items ${className}`} data-node={node.id} aria-busy="true" />
  if (!cart.cart.items.length) return <p className={`cos-muted ${className}`} data-node={node.id}>{props.empty_message || 'Your cart is empty.'}</p>
  return (
    <ul className={`cos-cart-items ${className}`} data-node={node.id} aria-label="Cart items">
      {cart.cart.items.map(item => (
        <li key={item.sku_code} className="cos-cart-line">
          <span className="cos-cart-name">{item.product_name}</span>
          <span className="cos-quantity" role="group" aria-label={`Quantity of ${item.product_name}`}>
            <button type="button" aria-label="Fewer" disabled={cart.busy} onClick={() => void cart.setQuantity(item.sku_code, item.quantity - 1)}>−</button>
            <span>{item.quantity}</span>
            <button type="button" aria-label="More" disabled={cart.busy} onClick={() => void cart.setQuantity(item.sku_code, item.quantity + 1)}>+</button>
          </span>
          <span>{formatPrice(item.price_minor_units * item.quantity, item.currency_code)}</span>
          <button type="button" className="cos-link" disabled={cart.busy} onClick={() => void cart.setQuantity(item.sku_code, 0)}>Remove</button>
        </li>
      ))}
      {cart.error && <li role="alert" className="cos-error">{cart.error}</li>}
    </ul>
  )
}

export function CartSummaryView({ node, props, className }: RendererProps<CartSummaryProps>) {
  const cart = useCart()
  const [code, setCode] = useState('')
  const c = cart?.cart
  const currency = c?.items[0]?.currency_code
  const total = c ? c.subtotal_minor_units - c.discount_minor_units : 0
  return (
    <div className={`cos-cart-summary ${className}`} data-node={node.id}>
      <dl>
        <div><dt>Subtotal</dt><dd>{c && currency ? formatPrice(c.subtotal_minor_units, currency) : '—'}</dd></div>
        {!!c?.discount_minor_units && currency && <div><dt>Discount</dt><dd>−{formatPrice(c.discount_minor_units, currency)}</dd></div>}
        <div className="cos-total"><dt>Total</dt><dd>{c && currency ? formatPrice(total, currency) : '—'}</dd></div>
      </dl>
      {props.show_coupon !== false && c && c.items.length > 0 && (
        <form className="cos-coupon" onSubmit={e => { e.preventDefault(); void cart!.applyCoupon(code.trim()) }}>
          <label htmlFor={`${node.id}-coupon`}>Coupon code</label>
          <input id={`${node.id}-coupon`} value={code} onChange={e => setCode(e.target.value)} placeholder={c.coupon_code ?? ''} />
          <button type="submit" disabled={cart!.busy}>Apply</button>
        </form>
      )}
    </div>
  )
}

export function CheckoutButtonView({ node, props, className }: RendererProps<CheckoutButtonProps>) {
  const cart = useCart()
  const env = useRenderEnv()
  const empty = !cart?.cart?.items.length
  const live = env.mode === 'live' && !empty
  return (
    <a className={`cos-button cos-button-primary ${className}`} data-node={node.id} href={live ? env.storePath('checkout') : undefined}
      aria-disabled={live ? undefined : true} role={live ? undefined : 'link'}>
      {props.label || 'Checkout'}
    </a>
  )
}
