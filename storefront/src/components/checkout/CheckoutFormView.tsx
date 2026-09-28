import { useEffect, useState } from 'react'
import { useStorefrontAPI } from '../../api/ApiContext'
import type { Address, ShippingOption } from '../../api/storefront'
import { formatPrice } from '../../core/money'
import { useRenderEnv } from '../../runtime/RenderContext'
import { useCart } from '../../runtime/page/CartContext'
import { useCustomer } from '../../runtime/page/CustomerContext'
import { rememberOrder } from '../../runtime/page/OrderContext'
import type { RendererProps } from '../types'
import { complete, deliverable, emptyAddress, safePaymentURL } from './address'
import type { CheckoutFormProps } from './checkout'

const FIELDS: { key: keyof Address; label: string; autoComplete: string; required?: boolean; wide?: boolean }[] = [
  { key: 'first_name', label: 'First name', autoComplete: 'given-name', required: true },
  { key: 'last_name', label: 'Last name', autoComplete: 'family-name', required: true },
  { key: 'address1', label: 'Street and number', autoComplete: 'address-line1', required: true, wide: true },
  { key: 'address2', label: 'Apartment, suite (optional)', autoComplete: 'address-line2', wide: true },
  { key: 'postcode', label: 'Postcode', autoComplete: 'postal-code', required: true },
  { key: 'city', label: 'City', autoComplete: 'address-level2', required: true },
  { key: 'phone', label: 'Phone (optional)', autoComplete: 'tel' },
]

const COUNTRIES = [
  { code: 'NL', name: 'Netherlands' }, { code: 'BE', name: 'Belgium' },
  { code: 'DE', name: 'Germany' }, { code: 'FR', name: 'France' },
]

/** Delivery pricing waits for the shopper to pause typing. */
const PRICE_DELAY_MS = 400

function useShippingOptions(cartToken: string | undefined, address: Address) {
  const api = useStorefrontAPI()
  const [options, setOptions] = useState<ShippingOption[]>([])
  const key = `${address.address1}|${address.postcode}|${address.city}|${address.country}`
  useEffect(() => {
    if (!cartToken || !deliverable(address)) return
    let cancelled = false
    const t = setTimeout(() => {
      api.shippingOptions(cartToken, address).then(o => !cancelled && setOptions(o), () => !cancelled && setOptions([]))
    }, PRICE_DELAY_MS)
    return () => { cancelled = true; clearTimeout(t) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [api, cartToken, key])
  return options
}

/** Hosted-payment checkout: places the order, then sends the shopper to the
 *  payment provider's page (or straight to the order page when nothing is due). */
export function CheckoutFormView({ node, props, className }: RendererProps<CheckoutFormProps>) {
  const api = useStorefrontAPI()
  const env = useRenderEnv()
  const cart = useCart()
  const customer = useCustomer()
  const [email, setEmail] = useState('')
  const [address, setAddress] = useState<Address>(emptyAddress)
  const [shipping, setShipping] = useState('')
  const [placing, setPlacing] = useState(false)
  const [error, setError] = useState('')
  const token = cart?.cart?.token
  const options = useShippingOptions(token, address)

  const accountEmail = customer?.account?.email
  useEffect(() => {
    if (accountEmail) setEmail(e => e || accountEmail)
  }, [accountEmail])

  useEffect(() => {
    setShipping(s => (options.some(o => o.title === s) ? s : options[0]?.title ?? ''))
  }, [options])

  if (env.mode === 'editor') {
    return <div className={`cos-placeholder ${className}`} data-node={node.id}>Checkout: contact, address, delivery choice and payment.</div>
  }
  if (!cart?.cart) return <div className={className} data-node={node.id} aria-busy="true" />
  if (!cart.cart.items.length) {
    return <p className={className} data-node={node.id}>Your cart is empty. <a href={env.storePath('')}>Continue shopping</a></p>
  }

  const current = cart.cart
  const set = (key: keyof Address, value: string) => setAddress(a => ({ ...a, [key]: value }))
  const currency = current.items[0]?.currency_code
  const chosen = options.find(o => o.title === shipping)
  const total = current.subtotal_minor_units - current.discount_minor_units + (chosen?.amount_minor_units ?? 0)

  const place = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!complete(address)) return
    setPlacing(true)
    setError('')
    try {
      const result = await api.checkout(current.token, { email, billing: address, shipping: address, shipping_method_title: shipping || undefined })
      rememberOrder(result.order_number, email)
      cart.clear()
      env.navigate(safePaymentURL(result.payment_url) ?? env.storePath(`orders/${encodeURIComponent(result.order_number)}`))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Your order could not be placed')
      setPlacing(false)
    }
  }

  return (
    <form className={`cos-checkout ${className}`} data-node={node.id} onSubmit={e => void place(e)}>
      <fieldset>
        <legend className="cos-label">Contact</legend>
        <label htmlFor={`${node.id}-email`}>Email</label>
        <input id={`${node.id}-email`} type="email" required autoComplete="email" value={email} onChange={e => setEmail(e.target.value)} />
      </fieldset>
      <fieldset className="cos-checkout-address">
        <legend className="cos-label">Delivery address</legend>
        {FIELDS.map(f => (
          <div key={f.key} className={f.wide ? 'cos-wide' : undefined}>
            <label htmlFor={`${node.id}-${f.key}`}>{f.label}</label>
            <input id={`${node.id}-${f.key}`} required={f.required} autoComplete={f.autoComplete} value={address[f.key] ?? ''} onChange={e => set(f.key, e.target.value)} />
          </div>
        ))}
        <div>
          <label htmlFor={`${node.id}-country`}>Country</label>
          <select id={`${node.id}-country`} autoComplete="country" value={address.country} onChange={e => set('country', e.target.value)}>
            {COUNTRIES.map(c => <option key={c.code} value={c.code}>{c.name}</option>)}
          </select>
        </div>
      </fieldset>
      {options.length > 0 && (
        <fieldset>
          <legend className="cos-label">Delivery</legend>
          {options.map(o => (
            <label key={o.title} className="cos-radio">
              <input type="radio" name={`${node.id}-shipping`} checked={shipping === o.title} onChange={() => setShipping(o.title)} />
              {`${o.title} — ${formatPrice(o.amount_minor_units, o.currency_code)}`}
            </label>
          ))}
        </fieldset>
      )}
      <div className="cos-checkout-total">
        <span>Total</span>
        <strong>{currency ? formatPrice(total, currency) : ''}</strong>
      </div>
      {error && <p role="alert" className="cos-error">{error}</p>}
      <button type="submit" className="cos-button cos-button-primary" disabled={placing || !email || !complete(address)}>
        {placing ? 'Placing order…' : props.button_label || 'Place order'}
      </button>
      <p className="cos-muted">You'll be taken to our payment provider to pay securely.</p>
    </form>
  )
}
