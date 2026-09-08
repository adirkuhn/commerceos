import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useCart } from '../../cart/CartContext'
import { getShippingOptions } from '../../api/cart'
import { formatMoney } from '../../format'
import type { CheckoutAddress, ShippingOption } from '../../types'

function emptyAddress(): CheckoutAddress {
  return { first_name: '', last_name: '', address1: '', city: '', postcode: '', country: '' }
}

function isAddressComplete(a: CheckoutAddress): boolean {
  return Boolean(a.address1 && a.city && a.postcode && a.country)
}

export function CheckoutFormBlock() {
  const { cart, checkout, error } = useCart()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [address, setAddress] = useState<CheckoutAddress>(emptyAddress())
  const [shippingOptions, setShippingOptions] = useState<ShippingOption[]>([])
  const [shippingOptionsLoading, setShippingOptionsLoading] = useState(false)
  const [selectedTitle, setSelectedTitle] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const cartToken = cart?.token

  // Debounced: re-quotes shipping every time the address changes enough
  // to matter, not on every keystroke. An address MyParcel can't quote
  // (unsupported country, no house number, no connection at all -- see
  // GetShippingOptionsUseCase's own doc comment) just yields an empty
  // list -- checkout still works, with shipping cost communicated some
  // other way.
  useEffect(() => {
    if (!cartToken || !isAddressComplete(address)) {
      setShippingOptions([])
      setSelectedTitle(null)
      return
    }
    let cancelled = false
    setShippingOptionsLoading(true)
    const timer = setTimeout(() => {
      getShippingOptions(cartToken, address)
        .then(options => {
          if (cancelled) return
          setShippingOptions(options)
          setSelectedTitle(options[0]?.title ?? null)
        })
        .catch(() => { if (!cancelled) setShippingOptions([]) })
        .finally(() => { if (!cancelled) setShippingOptionsLoading(false) })
    }, 500)
    return () => { cancelled = true; clearTimeout(timer) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cartToken, address.address1, address.city, address.postcode, address.country])

  if (!cart || cart.items.length === 0) {
    return <p className="p-6 text-center text-brand-500">Your cart is empty.</p>
  }

  const set = (key: keyof CheckoutAddress, value: string) => setAddress(a => ({ ...a, [key]: value }))

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setFormError(null)
    setSubmitting(true)
    try {
      const result = await checkout({ email, billing: address, shipping: address, shipping_method_title: selectedTitle ?? undefined })
      navigate(`/order-confirmation/${result.order_number}`, { state: { email } })
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Could not place your order')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="mx-auto max-w-md space-y-3 p-6">
      {(formError ?? error) && <p className="text-sm text-red-600">{formError ?? error}</p>}
      <Field label="Email" type="email" value={email} onChange={setEmail} />
      <Field label="First name" value={address.first_name} onChange={v => set('first_name', v)} />
      <Field label="Last name" value={address.last_name} onChange={v => set('last_name', v)} />
      <Field label="Address" value={address.address1} onChange={v => set('address1', v)} />
      <Field label="City" value={address.city} onChange={v => set('city', v)} />
      <Field label="Postcode" value={address.postcode} onChange={v => set('postcode', v)} />
      <Field label="Country" value={address.country} onChange={v => set('country', v)} />

      {shippingOptionsLoading && <p className="text-xs text-gray-400">Checking shipping options…</p>}
      {!shippingOptionsLoading && shippingOptions.length > 0 && (
        <fieldset className="space-y-1.5">
          <legend className="mb-1 text-sm text-brand-500">Shipping</legend>
          {shippingOptions.map(option => (
            <label key={option.title} className="flex items-center justify-between rounded-lg border border-gray-200 px-3 py-2 text-sm">
              <span className="flex items-center gap-2">
                <input
                  type="radio"
                  name="shipping-option"
                  checked={selectedTitle === option.title}
                  onChange={() => setSelectedTitle(option.title)}
                />
                {option.title}
              </span>
              <span className="text-brand-500">{formatMoney(option.amount_minor_units, option.currency_code)}</span>
            </label>
          ))}
        </fieldset>
      )}

      <button
        type="submit"
        disabled={submitting}
        className="w-full rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-700 disabled:opacity-50"
      >
        {submitting ? 'Placing order…' : 'Place order'}
      </button>
    </form>
  )
}

function Field({ label, value, onChange, type = 'text' }: { label: string; value: string; onChange: (v: string) => void; type?: string }) {
  return (
    <label className="block text-sm">
      <span className="mb-1 block text-brand-500">{label}</span>
      <input
        type={type}
        value={value}
        required
        onChange={e => onChange(e.target.value)}
        className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-brand-500"
      />
    </label>
  )
}
