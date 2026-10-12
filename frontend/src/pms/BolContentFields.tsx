import { useTranslation } from 'react-i18next'
import type { UpsertBolProductContentPayload } from '../integrations/api'

export const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink outline-none focus:ring-1 focus:ring-focus'

export const labelClass = 'block text-2xs font-medium text-ink-4 caps-label mb-1.5'

export function defaultBolProductContent(): UpsertBolProductContentPayload {
  return {
    manufacturer_contact: '', manufacturer_address: '', manufacturer_name: '',
    shelf_life_closed: '', shelf_life_after_opening: '', ingredients: '',
    content_amount: '', content_unit: '', country_of_origin: '', substance: '',
    instruction_language: '', packaging_language: '',
    items_per_package: null, package_insert_url: '', is_gift: false, dermatologically_tested: false,
    hazard_statements: '', skin_type: '', hypoallergenic: false, contains_zinc_oxide: false,
    content_per_dispenser_amount: '', content_per_dispenser_unit: '', brand: '',
    fragrance_free: false, ph_neutral: false, cooling_effect: false,
    package_width_mm: null, package_height_mm: null, package_length_mm: null, package_weight_g: null,
    precaution_statements: '', precautions: '', warnings: '', video_url: '',
  }
}

export function ContentTextField({ id, label, value, onChange, multiline, hint }: {
  id: string; label: string; value: string; onChange: (v: string) => void; multiline?: boolean; hint?: string
}) {
  return (
    <div>
      <label htmlFor={id} className={labelClass}>{label}</label>
      {multiline ? (
        <textarea id={id} value={value} onChange={e => onChange(e.target.value)} rows={3} className={inputClass} />
      ) : (
        <input id={id} type="text" value={value} onChange={e => onChange(e.target.value)} className={inputClass} />
      )}
      {hint && <p className="text-2xs text-ink-4 mt-1">{hint}</p>}
    </div>
  )
}

// Bol only accepts a value from a fixed list for these attributes (LOV-
// constrained, see bolContentLovs.ts) -- free text here would just get
// silently dropped by Bol the same way a wrong attribute id used to.
export function ContentSelectField({ id, label, value, onChange, options, hint }: {
  id: string; label: string; value: string; onChange: (v: string) => void; options: string[]; hint?: string
}) {
  const { t } = useTranslation('pms')
  return (
    <div>
      <label htmlFor={id} className={labelClass}>{label}</label>
      <select id={id} value={value} onChange={e => onChange(e.target.value)} className={inputClass}>
        <option value="">{t('bolField.notSet')}</option>
        {options.map(o => <option key={o} value={o}>{o}</option>)}
      </select>
      {hint && <p className="text-2xs text-ink-4 mt-1">{hint}</p>}
    </div>
  )
}

export function ContentNumberField({ id, label, value, onChange, unit }: {
  id: string; label: string; value: number | null | undefined; onChange: (v: number | null) => void; unit?: string
}) {
  return (
    <div>
      <label htmlFor={id} className={labelClass}>{label}{unit ? ` (${unit})` : ''}</label>
      <input
        id={id} type="number" value={value ?? ''}
        onChange={e => onChange(e.target.value === '' ? null : Number(e.target.value))}
        className={inputClass}
      />
    </div>
  )
}

export function ContentCheckField({ id, label, checked, onChange }: {
  id: string; label: string; checked: boolean | null | undefined; onChange: (v: boolean) => void
}) {
  return (
    <label htmlFor={id} className="flex items-center gap-1.5 text-2xs text-ink-2">
      <input id={id} type="checkbox" checked={checked ?? false} onChange={e => onChange(e.target.checked)} />
      {label}
    </label>
  )
}
