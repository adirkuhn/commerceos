import { useState } from 'react'
import { ArrowLeft } from 'lucide-react'
import type { Customer, CustomerInput, Address } from './types'

// ── Customer Edit Page ───────────────────────────────────────────────────────

interface CustomerEditPageProps {
  customer: Customer | null
  onCreate: (input: CustomerInput) => Promise<Customer>
  onUpdate: (id: string, input: CustomerInput) => Promise<Customer>
  onClose: () => void
}

function emptyAddress(): Address {
  return { first_name: '', last_name: '', company: '', address_1: '', address_2: '', city: '', state: '', postcode: '', country: '', email: '', phone: '' }
}

export function CustomerEditPage({ customer, onCreate, onUpdate, onClose }: CustomerEditPageProps) {
  const [firstName, setFirstName] = useState(customer?.first_name ?? '')
  const [lastName, setLastName] = useState(customer?.last_name ?? '')
  const [email, setEmail] = useState(customer?.email ?? '')
  const [billing, setBilling] = useState<Address>({ ...emptyAddress(), ...customer?.billing })
  const [shipping, setShipping] = useState<Address>({ ...emptyAddress(), ...customer?.shipping })
  const [isPayingCustomer, setIsPayingCustomer] = useState(customer?.is_paying_customer ?? false)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  // A deep link (Cmd+K search, direct URL, hard refresh) mounts this page
  // before useCustomers' list fetch resolves, so the useState initializers
  // above run with customer still null. Once the list arrives and `customer`
  // is passed in for the first time, re-seed the form from it -- without
  // this, the fields stay blank forever since useState never re-runs.
  const [hydrated, setHydrated] = useState(false)
  if (!hydrated && customer) {
    setFirstName(customer.first_name ?? '')
    setLastName(customer.last_name ?? '')
    setEmail(customer.email ?? '')
    setBilling({ ...emptyAddress(), ...customer.billing })
    setShipping({ ...emptyAddress(), ...customer.shipping })
    setIsPayingCustomer(customer.is_paying_customer ?? false)
    setHydrated(true)
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      const input: CustomerInput = {
        email, first_name: firstName, last_name: lastName, billing, shipping, is_paying_customer: isPayingCustomer,
      }
      if (customer) {
        await onUpdate(customer.id, input)
      } else {
        await onCreate(input)
      }
      onClose()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save customer')
    } finally {
      setSaving(false)
    }
  }

  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Customer
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {customer ? [firstName, lastName].filter(Boolean).join(' ') || 'Edit customer' : 'New customer'}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
          >
            Discard
          </button>
          <button
            type="submit" form="customer-form" disabled={saving || !email.trim()}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : customer ? 'Save' : 'Create customer'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto p-6 space-y-5">
          <form id="customer-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border border-line p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-ink">General</h3>
                <label className="flex items-center gap-1.5 text-2xs text-ink-3">
                  <input type="checkbox" checked={isPayingCustomer} onChange={e => setIsPayingCustomer(e.target.checked)} />
                  Paying customer
                </label>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="customer-first-name" className={labelClass}>First name</label>
                  <input id="customer-first-name" type="text" autoFocus value={firstName} onChange={e => setFirstName(e.target.value)} className={inputClass} />
                </div>
                <div>
                  <label htmlFor="customer-last-name" className={labelClass}>Last name</label>
                  <input id="customer-last-name" type="text" value={lastName} onChange={e => setLastName(e.target.value)} className={inputClass} />
                </div>
              </div>
              <div>
                <label htmlFor="customer-email" className={labelClass}>Email</label>
                <input id="customer-email" type="email" required value={email} onChange={e => setEmail(e.target.value)} className={inputClass} />
              </div>
              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>

            <AddressFields title="Billing address" prefix="billing" value={billing} onChange={setBilling} showEmailPhone />
            <AddressFields title="Shipping address" prefix="shipping" value={shipping} onChange={setShipping} />
          </form>
        </div>
      </div>
    </div>
  )
}

function AddressFields({ title, prefix, value, onChange, showEmailPhone }: { title: string; prefix: string; value: Address; onChange: (a: Address) => void; showEmailPhone?: boolean }) {
  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'
  const set = (field: keyof Address) => (e: React.ChangeEvent<HTMLInputElement>) => onChange({ ...value, [field]: e.target.value })
  const id = (field: string) => `${prefix}-${field}`
  const label = (text: string) => `${title.replace(' address', '')} ${text}`

  return (
    <div className="rounded-xl border border-line p-4 space-y-3">
      <h3 className="text-xs font-semibold text-ink">{title}</h3>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor={id('address-1')} className={labelClass}>{label('address line 1')}</label>
          <input id={id('address-1')} type="text" value={value.address_1 ?? ''} onChange={set('address_1')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('address-2')} className={labelClass}>{label('address line 2')}</label>
          <input id={id('address-2')} type="text" value={value.address_2 ?? ''} onChange={set('address_2')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('city')} className={labelClass}>{label('city')}</label>
          <input id={id('city')} type="text" value={value.city ?? ''} onChange={set('city')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('state')} className={labelClass}>{label('state')}</label>
          <input id={id('state')} type="text" value={value.state ?? ''} onChange={set('state')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('postcode')} className={labelClass}>{label('postcode')}</label>
          <input id={id('postcode')} type="text" value={value.postcode ?? ''} onChange={set('postcode')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('country')} className={labelClass}>{label('country')}</label>
          <input id={id('country')} type="text" value={value.country ?? ''} onChange={set('country')} className={inputClass} />
        </div>
        {showEmailPhone && (
          <div>
            <label htmlFor={id('phone')} className={labelClass}>{label('phone')}</label>
            <input id={id('phone')} type="tel" value={value.phone ?? ''} onChange={set('phone')} className={inputClass} />
          </div>
        )}
      </div>
    </div>
  )
}
