import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { NavLink } from '../../components/site/site'
import { LinkField, TextField } from './fields'
import { moveItem, removeItem, replaceItem } from './listOps'

const LABEL_MAX = 40

/** Commits only non-blank text; the box may be empty while retyping. */
function LabelInput({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  const [draft, setDraft] = useState(value)
  useEffect(() => setDraft(value), [value])
  return <TextField label={label} value={draft} maxLength={LABEL_MAX} onChange={v => { setDraft(v); if (v.trim()) onChange(v) }} />
}

/** An ordered list of menu links: a label and a safe link each. */
export function LinksField({ label, value, max, onChange }: { label: string; value: NavLink[]; max: number; onChange: (v: NavLink[]) => void }) {
  const { t } = useTranslation('editor')
  // The contract requires a non-empty label, so new rows start with one.
  const newLink: NavLink = { label: t('links.newLink'), link: '/' }
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="text-xs font-medium text-ed-ink-2">{label} <span className="text-ed-ink-4">({t('links.chosenOf', { chosen: value.length, max })})</span></legend>
      <ol className="flex flex-col gap-2" aria-label={label}>
        {value.map((item, i) => (
          <li key={i} className="flex flex-col gap-1 rounded-ed border border-ed-line p-2">
            <LabelInput label={t('links.linkText', { number: i + 1 })} value={item.label} onChange={v => onChange(replaceItem(value, i, { ...item, label: v }))} />
            <LinkField label={t('links.linkTarget', { number: i + 1 })} value={item.link} onChange={v => onChange(replaceItem(value, i, { ...item, link: v }))} />
            <div className="flex justify-end gap-2 text-sm">
              <button type="button" aria-label={t('links.moveUp', { number: i + 1 })} disabled={i === 0} onClick={() => onChange(moveItem(value, i, -1))}>↑</button>
              <button type="button" aria-label={t('links.moveDown', { number: i + 1 })} disabled={i === value.length - 1} onClick={() => onChange(moveItem(value, i, 1))}>↓</button>
              <button type="button" aria-label={t('links.remove', { number: i + 1 })} onClick={() => onChange(removeItem(value, i))}>✕</button>
            </div>
          </li>
        ))}
      </ol>
      {value.length < max && (
        <button type="button" className="rounded-ed border px-2 py-1 text-sm" onClick={() => onChange([...value, newLink])}>{t('links.add')}</button>
      )}
    </fieldset>
  )
}
