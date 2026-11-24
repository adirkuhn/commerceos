import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { apiErrorText } from '../../api/apiErrorText'
import type { ShippingAddress, ShippingAddressInput } from '../types'

const FIELDS = ['label', 'first_name', 'last_name', 'company', 'address_1', 'address_2', 'postcode', 'city', 'state', 'country', 'phone'] as const
type Field = typeof FIELDS[number]
const WIDE: ReadonlySet<Field> = new Set(['label', 'company', 'address_1', 'address_2'])

// Adds or edits one shipping address; withDefaultChoice (adding) offers
// to make it the default.
export function ShippingAddressForm({ initial, withDefaultChoice, onSave, onCancel, onRemove }: {
  initial?: ShippingAddress
  withDefaultChoice?: boolean
  onSave: (input: ShippingAddressInput) => Promise<void>
  onCancel: () => void
  onRemove?: () => void
}) {
  const { t } = useTranslation(['customers', 'common'])
  const [form, setForm] = useState<Record<Field, string>>(() =>
    Object.fromEntries(FIELDS.map(f => [f, initial?.[f] ?? ''])) as Record<Field, string>)
  const [asDefault, setAsDefault] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const hasLine = form.address_1.trim() !== ''

  async function save() {
    setSaving(true)
    setError('')
    try {
      const input = Object.fromEntries(FIELDS.map(f => [f, form[f].trim()])) as Record<Field, string>
      await onSave(withDefaultChoice ? { ...input, is_default: asDefault } : input)
    } catch (err) {
      setError(apiErrorText(err) ?? t('shippingAddresses.failed'))
    } finally {
      setSaving(false)
    }
  }

  const input = 'mt-1 w-full px-2.5 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus normal-case'
  return (
    <div className="space-y-2">
      <div className="grid grid-cols-2 gap-2">
        {FIELDS.map(f => (
          <label key={f} className={`block text-2xs font-medium text-ink-4 caps-label ${WIDE.has(f) ? 'col-span-2' : ''}`}>
            {t(`shippingAddresses.fields.${f}`)}
            <input value={form[f]} type={f === 'phone' ? 'tel' : 'text'} autoFocus={f === 'label'} className={input}
              maxLength={f === 'label' ? 60 : undefined}
              onChange={e => setForm(prev => ({ ...prev, [f]: e.target.value }))} />
          </label>
        ))}
      </div>
      {withDefaultChoice && (
        <label className="flex items-center gap-1.5 text-2xs text-ink-3">
          <input type="checkbox" checked={asDefault} onChange={e => setAsDefault(e.target.checked)} />
          {t('shippingAddresses.asDefault')}
        </label>
      )}
      {error && <p role="alert" className="text-danger">{error}</p>}
      <div className="flex items-center gap-2">
        <button type="button" onClick={save} disabled={!hasLine || saving}
          className="px-2.5 py-1 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-40">
          {t('common:actions.save')}
        </button>
        <button type="button" onClick={onCancel} className="px-2.5 py-1 rounded-lg border border-line text-xs text-ink-2 hover:bg-hover">
          {t('common:actions.cancel')}
        </button>
        {onRemove && (
          <button type="button" onClick={onRemove} className="ml-auto px-2.5 py-1 rounded-lg text-xs text-danger hover:bg-hover">
            {t('shippingAddresses.remove')}
          </button>
        )}
      </div>
    </div>
  )
}
