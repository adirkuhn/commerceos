import { useState } from 'react'
import { ArrowLeft } from 'lucide-react'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import type { Customer, CustomerInput, Address } from './types'
import { apiErrorText } from '../api/apiErrorText'
import { CustomerIdentityFields } from './identity/CustomerIdentityFields'
import { customerName, emptyIdentity, identityComplete, identityFromCustomer, identityToInput, type IdentityDraft } from './identity/customerIdentity'

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
  const { t } = useTranslation(['customers', 'common'])
  const [identity, setIdentity] = useState<IdentityDraft>(() => customer ? identityFromCustomer(customer) : emptyIdentity('private'))
  const [billing, setBilling] = useState<Address>({ ...emptyAddress(), ...customer?.billing })
  const [shipping, setShipping] = useState<Address>({ ...emptyAddress(), ...customer?.shipping })
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  // A deep link (Cmd+K search, direct URL, hard refresh) mounts this page
  // before useCustomers' list fetch resolves, so the useState initializers
  // above run with customer still null. Once the list arrives and `customer`
  // is passed in for the first time, re-seed the form from it -- without
  // this, the fields stay blank forever since useState never re-runs.
  const [hydrated, setHydrated] = useState(false)
  if (!hydrated && customer) {
    setIdentity(identityFromCustomer(customer))
    setBilling({ ...emptyAddress(), ...customer.billing })
    setShipping({ ...emptyAddress(), ...customer.shipping })
    setHydrated(true)
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      // A new company's person becomes its first contact person; the
      // billing address names the company unless it already names one.
      const billingCompany = identity.type === 'company' && !billing.company?.trim() ? { company: identity.company_name.trim() } : {}
      const input: CustomerInput = {
        ...identityToInput(identity, { withContactPerson: !customer }),
        billing: { ...billing, ...billingCompany }, shipping,
      }
      if (customer) {
        await onUpdate(customer.id, input)
      } else {
        await onCreate(input)
      }
      onClose()
    } catch (err) {
      setFormError(apiErrorText(err) ?? t('edit.failed'))
    } finally {
      setSaving(false)
    }
  }


  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> {t('edit.back')}
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {customer ? customerName(customer) || t('edit.editTitle') : t('edit.newTitle')}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
          >
            {t('edit.discard')}
          </button>
          <button
            type="submit" form="customer-form" disabled={saving || !(identityComplete(identity) || (identity.type === 'private' && billing.phone?.trim()))}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? t('common:actions.saving') : customer ? t('common:actions.save') : t('edit.create')}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto p-6 space-y-5">
          <form id="customer-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border border-line p-4 space-y-3">
              <h3 className="text-xs font-semibold text-ink">{t('edit.general')}</h3>
              <CustomerIdentityFields idPrefix="customer" value={identity} onChange={setIdentity} showContactPerson={!customer} />
              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>

            <AddressFields kind="billing" prefix="billing" value={billing} onChange={setBilling} showEmailPhone />
            <AddressFields kind="shipping" prefix="shipping" value={shipping} onChange={setShipping} />
          </form>
        </div>
      </div>
    </div>
  )
}

interface AddressLabels { title: string; address1: string; address2: string; city: string; state: string; postcode: string; country: string; phone: string }

function addressLabels(kind: 'billing' | 'shipping'): AddressLabels {
  return kind === 'billing'
    ? { title: i18next.t('customers:edit.billing.title'), address1: i18next.t('customers:edit.billing.address1'), address2: i18next.t('customers:edit.billing.address2'), city: i18next.t('customers:edit.billing.city'), state: i18next.t('customers:edit.billing.state'), postcode: i18next.t('customers:edit.billing.postcode'), country: i18next.t('customers:edit.billing.country'), phone: i18next.t('customers:edit.billing.phone') }
    : { title: i18next.t('customers:edit.shipping.title'), address1: i18next.t('customers:edit.shipping.address1'), address2: i18next.t('customers:edit.shipping.address2'), city: i18next.t('customers:edit.shipping.city'), state: i18next.t('customers:edit.shipping.state'), postcode: i18next.t('customers:edit.shipping.postcode'), country: i18next.t('customers:edit.shipping.country'), phone: i18next.t('customers:edit.shipping.phone') }
}

function AddressFields({ kind, prefix, value, onChange, showEmailPhone }: { kind: 'billing' | 'shipping'; prefix: string; value: Address; onChange: (a: Address) => void; showEmailPhone?: boolean }) {
  useTranslation('customers') // re-render when the language changes
  const labels = addressLabels(kind)
  const labelClass = 'block text-2xs font-medium text-ink-4 caps-label mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'
  const set = (field: keyof Address) => (e: React.ChangeEvent<HTMLInputElement>) => onChange({ ...value, [field]: e.target.value })
  const id = (field: string) => `${prefix}-${field}`

  return (
    <div className="rounded-xl border border-line p-4 space-y-3">
      <h3 className="text-xs font-semibold text-ink">{labels.title}</h3>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor={id('address-1')} className={labelClass}>{labels.address1}</label>
          <input id={id('address-1')} type="text" value={value.address_1 ?? ''} onChange={set('address_1')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('address-2')} className={labelClass}>{labels.address2}</label>
          <input id={id('address-2')} type="text" value={value.address_2 ?? ''} onChange={set('address_2')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('city')} className={labelClass}>{labels.city}</label>
          <input id={id('city')} type="text" value={value.city ?? ''} onChange={set('city')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('state')} className={labelClass}>{labels.state}</label>
          <input id={id('state')} type="text" value={value.state ?? ''} onChange={set('state')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('postcode')} className={labelClass}>{labels.postcode}</label>
          <input id={id('postcode')} type="text" value={value.postcode ?? ''} onChange={set('postcode')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('country')} className={labelClass}>{labels.country}</label>
          <input id={id('country')} type="text" value={value.country ?? ''} onChange={set('country')} className={inputClass} />
        </div>
        {showEmailPhone && (
          <div>
            <label htmlFor={id('phone')} className={labelClass}>{labels.phone}</label>
            <input id={id('phone')} type="tel" value={value.phone ?? ''} onChange={set('phone')} className={inputClass} />
          </div>
        )}
      </div>
    </div>
  )
}
