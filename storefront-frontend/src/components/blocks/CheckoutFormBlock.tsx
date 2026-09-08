import { useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useCart } from '../../cart/CartContext'
import type { CheckoutAddress } from '../../types'

function emptyAddress(): CheckoutAddress {
  return { first_name: '', last_name: '', address1: '', city: '', postcode: '', country: '' }
}

export function CheckoutFormBlock() {
  const { cart, checkout, error } = useCart()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [address, setAddress] = useState<CheckoutAddress>(emptyAddress())
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  if (!cart || cart.items.length === 0) {
    return <p className="p-6 text-center text-brand-500">Your cart is empty.</p>
  }

  const set = (key: keyof CheckoutAddress, value: string) => setAddress(a => ({ ...a, [key]: value }))

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setFormError(null)
    setSubmitting(true)
    try {
      const result = await checkout({ email, billing: address, shipping: address })
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
