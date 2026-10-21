import clsx from 'clsx'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { countryOptions } from '../../settings/company/countries'
import type { AddressDraft, AddressErrors, AddressField } from './shippingAddressDraft'

interface ShippingAddressFieldsProps {
  // Keeps input ids unique when the sender and return forms are both open.
  idPrefix: string
  draft: AddressDraft
  errors: AddressErrors
  disabled: boolean
  onChange: (field: AddressField, value: string) => void
}

// Text fields in display order, with the grid width each takes.
const TEXT_FIELDS: readonly { field: Exclude<AddressField, 'cc'>; wide?: boolean; type?: string; autoComplete?: string }[] = [
  { field: 'company', wide: true, autoComplete: 'organization' },
  { field: 'street', wide: true, autoComplete: 'address-line1' },
  { field: 'number' },
  { field: 'number_suffix' },
  { field: 'postal_code', autoComplete: 'postal-code' },
  { field: 'city', autoComplete: 'address-level2' },
  { field: 'email', type: 'email', autoComplete: 'email' },
  { field: 'phone', type: 'tel', autoComplete: 'tel' },
]

const inputClass = 'w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border text-ink disabled:opacity-60'

// The fields of one sender/return address, with inline errors.
export function ShippingAddressFields({ idPrefix, draft, errors, disabled, onChange }: ShippingAddressFieldsProps) {
  const { t } = useTranslation('shipping')

  function wrap(field: AddressField, wide: boolean | undefined, control: (id: string, describedBy: string | undefined) => ReactNode) {
    const id = `${idPrefix}-${field}`
    const error = errors[field]
    return (
      <div key={field} className={clsx(wide && 'sm:col-span-2')}>
        <label htmlFor={id} className="block text-2xs font-medium text-ink-3 mb-1">{t(`addresses.field.${field}`)}</label>
        {control(id, error ? `${id}-error` : undefined)}
        {error && <p id={`${id}-error`} className="mt-0.5 text-2xs text-danger">{t(`addresses.errors.${error}`)}</p>}
      </div>
    )
  }

  return (
    <div className="grid grid-cols-1 sm:grid-cols-4 gap-2 mt-2">
      {TEXT_FIELDS.map(({ field, wide, type, autoComplete }) => wrap(field, wide, (id, describedBy) => (
        <input
          id={id} type={type ?? 'text'} value={draft[field]} autoComplete={autoComplete} disabled={disabled}
          onChange={e => onChange(field, e.target.value)}
          aria-invalid={errors[field] ? true : undefined} aria-describedby={describedBy}
          className={clsx(inputClass, errors[field] ? 'border-danger' : 'border-line')}
        />
      )))}
      {wrap('cc', true, (id, describedBy) => (
        <select
          id={id} value={draft.cc.toUpperCase()} autoComplete="country" disabled={disabled}
          onChange={e => onChange('cc', e.target.value)}
          aria-invalid={errors.cc ? true : undefined} aria-describedby={describedBy}
          className={clsx(inputClass, errors.cc ? 'border-danger' : 'border-line')}
        >
          <option value="">{t('addresses.field.chooseCountry')}</option>
          {countryOptions(draft.cc.toUpperCase()).map(c => <option key={c.code} value={c.code}>{c.name}</option>)}
        </select>
      ))}
    </div>
  )
}
