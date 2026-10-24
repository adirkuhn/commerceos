import { useTranslation } from 'react-i18next'
import type { Address } from '../types'

const ROWS: { field: keyof Address; key: string; span?: boolean; type?: string }[] = [
  { field: 'first_name', key: 'firstName' }, { field: 'last_name', key: 'lastName' },
  { field: 'company', key: 'company', span: true },
  { field: 'address_1', key: 'street', span: true }, { field: 'address_2', key: 'street2', span: true },
  { field: 'postcode', key: 'postcode' }, { field: 'city', key: 'city' },
  { field: 'country', key: 'country' }, { field: 'phone', key: 'phone', type: 'tel' },
  { field: 'email', key: 'email', type: 'email', span: true },
]

export function AddressFields({ prefix, value, onChange }: { prefix: string; value: Address; onChange: (a: Address) => void }) {
  const { t } = useTranslation('orders')
  return (
    <div className="grid grid-cols-2 gap-x-3 gap-y-2.5">
      {ROWS.map(r => (
        <div key={r.field} className={r.span ? 'col-span-2' : ''}>
          <label htmlFor={`${prefix}-${r.field}`} className="block text-2xs font-medium text-ink-4 mb-1">{t(`form.${r.key}` as 'form.city')}</label>
          <input id={`${prefix}-${r.field}`} type={r.type ?? 'text'} value={value[r.field] ?? ''}
            onChange={e => onChange({ ...value, [r.field]: e.target.value })}
            className="w-full px-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus" />
        </div>
      ))}
    </div>
  )
}
