import { useEffect, useState } from 'react'
import { UserPlus, UserRound, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { apiErrorText } from '../../api/apiErrorText'
import { getCustomer } from '../../customers/api'
import { CustomerPicker } from '../../customers/CustomerPicker'
import type { ContactPerson, Customer } from '../../customers/types'
import { useContactPersons } from '../../customers/hooks/useContactPersons'
import { NewCustomerDialog } from './NewCustomerDialog'
import { customerName } from '../../customers/identity/customerIdentity'

const displayName = (c: Customer) => customerName(c) || c.billing?.phone || c.id

// Picks an existing customer or creates one; the page copies its addresses.
// For a company it also picks who the order is for (onContactPerson): its
// first contact person, or the one chosen.
export function OrderCustomerCard({ token, customer, onChange, onContactPerson }: {
  token: string
  customer: Customer | null
  onChange: (c: Customer | null) => void
  onContactPerson?: (p: ContactPerson | null) => void
}) {
  const { t } = useTranslation(['orders', 'customers'])
  const [creating, setCreating] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const company = customer?.customer_type === 'company'
  const contacts = useContactPersons(token, company && onContactPerson ? customer?.id ?? null : null)
  const [contactId, setContactId] = useState('')

  // A newly loaded company starts on its main (first) contact person.
  const firstContact = contacts.persons[0]
  useEffect(() => {
    if (!firstContact) return
    setContactId(firstContact.id)
    onContactPerson?.(firstContact)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [firstContact?.id])

  function chooseContact(id: string) {
    setContactId(id)
    onContactPerson?.(contacts.persons.find(p => p.id === id) ?? null)
  }

  async function pick(id: string) {
    setLoading(true)
    setError('')
    try {
      onChange(await getCustomer(token, id))
    } catch (e) {
      setError(apiErrorText(e) ?? t('detail.customerChangeFailed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="rounded-xl border border-line bg-surface p-4 space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-xs font-semibold text-ink">{t('form.customer')}</h3>
        {!customer && (
          <button type="button" onClick={() => setCreating(true)} className="flex items-center gap-1 text-2xs font-medium text-accent hover:underline">
            <UserPlus size={12} /> {t('form.newCustomer')}
          </button>
        )}
      </div>
      {customer ? (
        <div className="flex items-start gap-3 p-3 rounded-lg bg-sunken">
          <UserRound size={16} className="text-ink-4 mt-0.5" />
          <div className="flex-1 min-w-0 text-xs">
            <p className="font-medium text-ink">{displayName(customer)}</p>
            {customer.email && <p className="text-ink-3">{customer.email}</p>}
            {customer.billing?.city && <p className="text-ink-4">{[customer.billing.address_1, customer.billing.postcode, customer.billing.city].filter(Boolean).join(', ')}</p>}
          </div>
          <button type="button" onClick={() => onChange(null)} aria-label={t('form.removeCustomer')} title={t('form.removeCustomer')}
            className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover text-ink-4"><X size={12} /></button>
        </div>
      ) : null}
      {customer && company && contacts.persons.length > 0 && (
        <label className="block text-2xs font-medium text-ink-4 caps-label">
          {t('form.contactPerson')}
          <select value={contactId} onChange={e => chooseContact(e.target.value)}
            className="mt-1 w-full px-2.5 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus normal-case">
            {contacts.persons.map(p => (
              <option key={p.id} value={p.id}>{[p.first_name, p.last_name].filter(Boolean).join(' ')}{p.role ? ` — ${p.role}` : ''}</option>
            ))}
          </select>
        </label>
      )}
      {contacts.error && <p role="alert" className="text-2xs text-danger">{contacts.error}</p>}
      {customer ? null : (
        <div className="space-y-1.5">
          <CustomerPicker token={token} value={null} onChange={c => { if (c) void pick(c.id) }} />
          <p className="text-2xs text-ink-4">{loading ? t('form.loadingCustomer') : t('form.noCustomer')}</p>
        </div>
      )}
      {error && <p role="alert" className="text-2xs text-danger">{error}</p>}
      {creating && <NewCustomerDialog token={token} onClose={() => setCreating(false)} onCreated={c => { setCreating(false); onChange(c) }} />}
    </section>
  )
}
