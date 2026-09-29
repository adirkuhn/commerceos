import { useEffect, useId, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { safeHref } from '../../core/richText'

const input = 'w-full rounded-md border border-neutral-300 px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-600'

export function Field({ label, children, hint }: { label: string; children: (id: string) => ReactNode; hint?: string }) {
  const id = useId()
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="text-xs font-medium text-neutral-700">{label}</label>
      {children(id)}
      {hint && <p className="text-xs text-neutral-500">{hint}</p>}
    </div>
  )
}

export function TextField({ label, value, onChange, maxLength, multiline }: {
  label: string; value: string; onChange: (v: string) => void; maxLength?: number; multiline?: boolean
}) {
  return (
    <Field label={label}>
      {id => multiline
        ? <textarea id={id} className={`${input} min-h-28`} value={value} maxLength={maxLength} onChange={e => onChange(e.target.value)} />
        : <input id={id} className={input} value={value} maxLength={maxLength} onChange={e => onChange(e.target.value)} />}
    </Field>
  )
}

export function NumberField({ label, value, onChange, min, max }: { label: string; value: number; onChange: (v: number) => void; min?: number; max?: number }) {
  return (
    <Field label={label}>
      {id => (
        <input id={id} type="number" className={input} value={value} min={min} max={max}
          onChange={e => {
            const n = Math.round(Number(e.target.value))
            if (Number.isFinite(n)) onChange(Math.min(max ?? n, Math.max(min ?? n, n)))
          }} />
      )}
    </Field>
  )
}

export function SelectField<T extends string | number>({ label, value, options, onChange, allowEmpty }: {
  label: string; value: T | ''; options: { value: T; label: string }[]; onChange: (v: T | '') => void; allowEmpty?: string
}) {
  return (
    <Field label={label}>
      {id => (
        <select id={id} className={input} value={String(value)}
          onChange={e => {
            const raw = e.target.value
            const match = options.find(o => String(o.value) === raw)
            onChange(match ? match.value : '')
          }}>
          {allowEmpty !== undefined && <option value="">{allowEmpty}</option>}
          {options.map(o => <option key={String(o.value)} value={String(o.value)}>{o.label}</option>)}
        </select>
      )}
    </Field>
  )
}

export function ToggleField({ label, value, onChange }: { label: string; value: boolean; onChange: (v: boolean) => void }) {
  const id = useId()
  return (
    <div className="flex items-center justify-between gap-2">
      <label htmlFor={id} className="text-xs font-medium text-neutral-700">{label}</label>
      <input id={id} type="checkbox" role="switch" checked={value} onChange={e => onChange(e.target.checked)} />
    </div>
  )
}

/** Store paths ("/pages/about"), web links, mailto: and tel: only. The
 *  typed text is only committed once it's a valid link (or empty). */
export function LinkField({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  const [draft, setDraft] = useState(value)
  useEffect(() => setDraft(value), [value])
  const invalid = draft !== '' && !safeHref(draft)
  const { t } = useTranslation('editor')
  return (
    <Field label={label} hint={invalid ? t('fields.linkHint') : undefined}>
      {id => <input id={id} className={input} value={draft} aria-invalid={invalid} placeholder={t('fields.linkPlaceholder')} maxLength={2000}
        onChange={e => {
          setDraft(e.target.value)
          if (e.target.value === '' || safeHref(e.target.value)) onChange(e.target.value)
        }} />}
    </Field>
  )
}
