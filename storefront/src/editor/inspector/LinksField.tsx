import { useEffect, useState } from 'react'
import type { NavLink } from '../../components/site/site'
import { LinkField, TextField } from './fields'
import { moveItem, removeItem, replaceItem } from './listOps'

/** The contract requires a non-empty label, so new rows start with one. */
const NEW_LINK: NavLink = { label: 'Link', link: '/' }
const LABEL_MAX = 40

/** Commits only non-blank text; the box may be empty while retyping. */
function LabelInput({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  const [draft, setDraft] = useState(value)
  useEffect(() => setDraft(value), [value])
  return <TextField label={label} value={draft} maxLength={LABEL_MAX} onChange={v => { setDraft(v); if (v.trim()) onChange(v) }} />
}

/** An ordered list of menu links: a label and a safe link each. */
export function LinksField({ label, value, max, onChange }: { label: string; value: NavLink[]; max: number; onChange: (v: NavLink[]) => void }) {
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="text-xs font-medium text-neutral-700">{label} <span className="text-neutral-500">({value.length} of {max})</span></legend>
      <ol className="flex flex-col gap-2" aria-label={label}>
        {value.map((item, i) => (
          <li key={i} className="flex flex-col gap-1 rounded border border-neutral-200 p-2">
            <LabelInput label={`Link ${i + 1} text`} value={item.label} onChange={v => onChange(replaceItem(value, i, { ...item, label: v }))} />
            <LinkField label={`Link ${i + 1} goes to`} value={item.link} onChange={v => onChange(replaceItem(value, i, { ...item, link: v }))} />
            <div className="flex justify-end gap-2 text-sm">
              <button type="button" aria-label={`Move link ${i + 1} up`} disabled={i === 0} onClick={() => onChange(moveItem(value, i, -1))}>↑</button>
              <button type="button" aria-label={`Move link ${i + 1} down`} disabled={i === value.length - 1} onClick={() => onChange(moveItem(value, i, 1))}>↓</button>
              <button type="button" aria-label={`Remove link ${i + 1}`} onClick={() => onChange(removeItem(value, i))}>✕</button>
            </div>
          </li>
        ))}
      </ol>
      {value.length < max && (
        <button type="button" className="rounded-md border px-2 py-1 text-sm" onClick={() => onChange([...value, NEW_LINK])}>Add link</button>
      )}
    </fieldset>
  )
}
