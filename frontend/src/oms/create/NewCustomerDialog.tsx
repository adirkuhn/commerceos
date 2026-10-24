import { useState } from 'react'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { apiErrorText } from '../../api/apiErrorText'
import { Dialog } from '../../components/common/Dialog'
import { createCustomer } from '../../customers/api'
import type { Customer } from '../../customers/types'

const FIELDS = ['first_name', 'last_name', 'company', 'email', 'phone', 'address_1', 'postcode', 'city', 'country'] as const
type Field = typeof FIELDS[number]

// A customer created right from the order form -- email optional; the
// address goes on both billing and shipping.
export function NewCustomerDialog({ token, onClose, onCreated }: { token: string; onClose: () => void; onCreated: (c: Customer) => void }) {
  const { t } = useTranslation(['orders', 'common'])
  const [form, setForm] = useState<Record<Field, string>>(() => ({
    first_name: '', last_name: '', company: '', email: '', phone: '', address_1: '', postcode: '', city: '', country: 'NL',
  }))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const identifiable = ['first_name', 'last_name', 'company', 'email', 'phone'].some(f => form[f as Field].trim() !== '')

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError('')
    const address = { ...form }
    try {
      onCreated(await createCustomer(token, {
        first_name: form.first_name, last_name: form.last_name, email: form.email, billing: address, shipping: address,
      }))
    } catch (err) {
      setError(apiErrorText(err) ?? t('newCustomer.failed'))
    } finally {
      setSaving(false)
    }
  }

  const input = 'mt-1 w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink outline-none focus:ring-1 focus:ring-focus'
  const field = (f: Field, span = '') => (
    <label key={f} className={`block text-2xs font-medium text-ink-4 caps-label ${span}`}>
      {t(`newCustomer.${f}`)}
      <input value={form[f]} onChange={e => setForm(prev => ({ ...prev, [f]: e.target.value }))}
        type={f === 'email' ? 'email' : f === 'phone' ? 'tel' : 'text'} className={`${input} normal-case`} />
    </label>
  )

  return (
    <Dialog onClose={onClose} label={t('newCustomer.title')} className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onClose} />
      <form onSubmit={submit} className="relative w-full max-w-lg bg-raised rounded-xl shadow-2xl border border-line p-5 space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-ink">{t('newCustomer.title')}</h3>
          <button type="button" onClick={onClose} aria-label={t('common:actions.close')}><X size={14} className="text-ink-4" /></button>
        </div>
        <div className="grid grid-cols-2 gap-3">
          {field('first_name')}{field('last_name')}
          {field('company', 'col-span-2')}
          {field('email')}{field('phone')}
          {field('address_1', 'col-span-2')}
          {field('postcode')}{field('city')}
          {field('country')}
        </div>
        <p className="text-2xs text-ink-4">{t('newCustomer.hint')}</p>
        {error && <p role="alert" className="text-xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2">
          <button type="button" onClick={onClose} className="px-3 py-1.5 rounded-lg border border-line text-xs text-ink-2 hover:bg-hover">{t('common:actions.cancel')}</button>
          <button type="submit" disabled={!identifiable || saving}
            className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-40">
            {t('newCustomer.submit')}
          </button>
        </div>
      </form>
    </Dialog>
  )
}
