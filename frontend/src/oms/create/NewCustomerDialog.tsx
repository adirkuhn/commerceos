import { useState } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { apiErrorText } from '../../api/apiErrorText'
import { Dialog } from '../../components/common/Dialog'
import { createCustomer } from '../../customers/api'
import { CustomerIdentityFields } from '../../customers/identity/CustomerIdentityFields'
import { emptyIdentity, identityComplete, identityToInput, type IdentityDraft } from '../../customers/identity/customerIdentity'
import type { Address, Customer } from '../../customers/types'

const ADDRESS_FIELDS = ['address_1', 'postcode', 'city', 'country'] as const
type AddressField = typeof ADDRESS_FIELDS[number]

// A customer created right from the order form -- a company by default (the
// person typed becomes its contact person), email optional; the address goes
// on both billing and shipping. Portalled out of the order <form>, and its
// submit stops there: React bubbles submit along the component tree, so it
// would otherwise submit the order too.
export function NewCustomerDialog({ token, onClose, onCreated }: { token: string; onClose: () => void; onCreated: (c: Customer) => void }) {
  const { t } = useTranslation(['orders', 'common'])
  const [identity, setIdentity] = useState<IdentityDraft>(() => emptyIdentity('company'))
  const [address, setAddress] = useState<Record<AddressField, string>>({ address_1: '', postcode: '', city: '', country: 'NL' })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    e.stopPropagation()
    setSaving(true)
    setError('')
    const fields: Address = {
      first_name: identity.first_name.trim(), last_name: identity.last_name.trim(),
      company: identity.type === 'company' ? identity.company_name.trim() : '',
      ...address, email: identity.email.trim(), phone: identity.phone.trim(),
    }
    try {
      onCreated(await createCustomer(token, { ...identityToInput(identity, { withContactPerson: true }), billing: fields, shipping: fields }))
    } catch (err) {
      setError(apiErrorText(err) ?? t('newCustomer.failed'))
    } finally {
      setSaving(false)
    }
  }

  const input = 'mt-1 w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink outline-none focus:ring-1 focus:ring-focus normal-case'
  const field = (f: AddressField, span = '') => (
    <label key={f} className={`block text-2xs font-medium text-ink-4 caps-label ${span}`}>
      {t(`newCustomer.${f}`)}
      <input value={address[f]} onChange={e => setAddress(prev => ({ ...prev, [f]: e.target.value }))} type="text" className={input} />
    </label>
  )

  return createPortal(
    <Dialog onClose={onClose} label={t('newCustomer.title')} className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onClose} />
      <form onSubmit={submit} className="relative w-full max-w-lg max-h-full overflow-y-auto bg-raised rounded-xl shadow-2xl border border-line p-5 space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-ink">{t('newCustomer.title')}</h3>
          <button type="button" onClick={onClose} aria-label={t('common:actions.close')}><X size={14} className="text-ink-4" /></button>
        </div>
        <CustomerIdentityFields idPrefix="new-customer" value={identity} onChange={setIdentity} showContactPerson showPhone />
        <fieldset className="space-y-2">
          <legend className="text-xs font-semibold text-ink">{t('newCustomer.address')}</legend>
          <div className="grid grid-cols-2 gap-3">
            {field('address_1', 'col-span-2')}
            {field('postcode')}{field('city')}
            {field('country')}
          </div>
        </fieldset>
        <p className="text-2xs text-ink-4">{t('newCustomer.hint')}</p>
        {error && <p role="alert" className="text-xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2">
          <button type="button" onClick={onClose} className="px-3 py-1.5 rounded-lg border border-line text-xs text-ink-2 hover:bg-hover">{t('common:actions.cancel')}</button>
          <button type="submit" disabled={!identityComplete(identity) || saving}
            className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-40">
            {t('newCustomer.submit')}
          </button>
        </div>
      </form>
    </Dialog>,
    document.body,
  )
}
