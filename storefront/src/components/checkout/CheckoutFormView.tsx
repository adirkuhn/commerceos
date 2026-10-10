import { useEffect, useMemo, useState } from 'react'
import type { TFunction } from 'i18next'
import { useTranslation } from 'react-i18next'
import { useStorefrontAPI } from '../../api/ApiContext'
import type { Address, ShippingOption } from '../../api/storefront'
import { useShopFormat } from '../../i18n/format'
import { shopperError } from '../../i18n/shopperError'
import { useRenderEnv } from '../../runtime/RenderContext'
import { useCart } from '../../runtime/page/CartContext'
import { useCustomer } from '../../runtime/page/CustomerContext'
import { rememberOrder } from '../../runtime/page/OrderContext'
import type { RendererProps } from '../types'
import { complete, deliverable, emptyAddress, safePaymentURL } from './address'
import type { CheckoutFormProps } from './checkout'

interface AddressField { key: keyof Address; label: string; autoComplete: string; required?: boolean; wide?: boolean }

const addressFields = (t: TFunction): AddressField[] => [
  { key: 'first_name', label: t('checkout.firstName'), autoComplete: 'given-name', required: true },
  { key: 'last_name', label: t('checkout.lastName'), autoComplete: 'family-name', required: true },
  { key: 'address1', label: t('checkout.street'), autoComplete: 'address-line1', required: true, wide: true },
  { key: 'address2', label: t('checkout.apartment'), autoComplete: 'address-line2', wide: true },
  { key: 'postcode', label: t('checkout.postcode'), autoComplete: 'postal-code', required: true },
  { key: 'city', label: t('checkout.city'), autoComplete: 'address-level2', required: true },
  { key: 'phone', label: t('checkout.phone'), autoComplete: 'tel' },
]

const COUNTRY_CODES = ['NL', 'BE', 'DE', 'FR']

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
  const { t, i18n } = useTranslation()
  const { price } = useShopFormat()
  const countries = useMemo(() => {
    const names = new Intl.DisplayNames([i18n.language], { type: 'region' })
    return COUNTRY_CODES.map(code => ({ code, name: names.of(code) ?? code }))
  }, [i18n.language])
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
    return <div className={`cos-placeholder ${className}`} data-node={node.id}>{t('hint.checkout')}</div>
  }
  if (!cart?.cart) return <div className={className} data-node={node.id} aria-busy="true"><p className="cos-muted">{t('hint.loading')}</p></div>
  if (!cart.cart.items.length) {
    return <p className={className} data-node={node.id}>{t('cart.empty')} <a href={env.storePath('')}>{t('common.continueShopping')}</a></p>
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
      setError(shopperError(err, i18n, t('checkout.failed')))
      setPlacing(false)
    }
  }

  return (
    <form className={`cos-checkout ${className}`} data-node={node.id} onSubmit={e => void place(e)}>
      <fieldset>
        <legend className="cos-label">{t('checkout.contact')}</legend>
        <label htmlFor={`${node.id}-email`}>{t('common.email')}</label>
        <input id={`${node.id}-email`} type="email" required autoComplete="email" value={email} onChange={e => setEmail(e.target.value)} />
      </fieldset>
      <fieldset className="cos-checkout-address">
        <legend className="cos-label">{t('checkout.deliveryAddress')}</legend>
        {addressFields(t).map(f => (
          <div key={f.key} className={f.wide ? 'cos-wide' : undefined}>
            <label htmlFor={`${node.id}-${f.key}`}>{f.label}</label>
            <input id={`${node.id}-${f.key}`} required={f.required} autoComplete={f.autoComplete} value={address[f.key] ?? ''} onChange={e => set(f.key, e.target.value)} />
          </div>
        ))}
        <div>
          <label htmlFor={`${node.id}-country`}>{t('checkout.country')}</label>
          <select id={`${node.id}-country`} autoComplete="country" value={address.country} onChange={e => set('country', e.target.value)}>
            {countries.map(c => <option key={c.code} value={c.code}>{c.name}</option>)}
          </select>
        </div>
      </fieldset>
      {options.length > 0 && (
        <fieldset>
          <legend className="cos-label">{t('checkout.delivery')}</legend>
          {options.map(o => (
            <label key={o.title} className="cos-radio">
              <input type="radio" name={`${node.id}-shipping`} checked={shipping === o.title} onChange={() => setShipping(o.title)} />
              {`${o.title} — ${price(o.amount_minor_units, o.currency_code)}`}
            </label>
          ))}
        </fieldset>
      )}
      <div className="cos-checkout-total">
        <span>{t('common.total')}</span>
        <strong>{currency ? price(total, currency) : ''}</strong>
      </div>
      {error && <p role="alert" className="cos-error">{error}</p>}
      <button type="submit" className="cos-button cos-button-primary" disabled={placing || !email || !complete(address)}>
        {placing ? t('checkout.placing') : props.button_label || t('checkout.placeOrder')}
      </button>
      <p className="cos-muted">{t('checkout.paymentNote')}</p>
    </form>
  )
}
