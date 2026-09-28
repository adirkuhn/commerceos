import { useEffect, useState } from 'react'
import type { ThemeSettings } from '../../core/document'
import { NumberField, SelectField, TextField, ToggleField } from './fields'
import { AssetPicker } from './pickers'

const SYSTEM_FONTS = [
  { value: 'system-sans', label: 'Sans-serif' }, { value: 'system-serif', label: 'Serif' },
  { value: 'system-rounded', label: 'Rounded' }, { value: 'system-mono', label: 'Monospace' },
]
const COLOR_NAMES: (keyof ThemeSettings['colors'])[] = ['primary', 'secondary', 'accent', 'text', 'muted', 'background', 'surface']

const assetRef = (id?: string | null) => (id ? `asset:${id}` : '')
const assetId = (ref: string) => ref.replace(/^asset:/, '') || null

/** Theme-wide design settings: brand, colours, fonts, layout, buttons, products. */
export function SettingsPanel({ settings, onSave, saving }: { settings: ThemeSettings; onSave: (s: ThemeSettings) => void; saving: boolean }) {
  const [draft, setDraft] = useState(settings)
  useEffect(() => setDraft(settings), [settings])
  const set = <K extends keyof ThemeSettings>(key: K, value: Partial<ThemeSettings[K]>) =>
    setDraft(d => ({ ...d, [key]: { ...d[key], ...value } }))
  const changed = JSON.stringify(draft) !== JSON.stringify(settings)
  const google = draft.typography.source === 'google'

  return (
    <div className="flex flex-col gap-5">
      <h2 className="font-semibold">Theme settings</h2>
      <section className="flex flex-col gap-2" aria-label="Brand">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-neutral-500">Brand</h3>
        <AssetPicker label="Logo" value={assetRef(draft.brand.logo_asset_id)} onChange={r => set('brand', { logo_asset_id: assetId(r) })} />
        <AssetPicker label="Favicon" value={assetRef(draft.brand.favicon_asset_id)} onChange={r => set('brand', { favicon_asset_id: assetId(r) })} />
      </section>
      <section className="grid grid-cols-2 gap-2" aria-label="Colours">
        <h3 className="col-span-2 text-xs font-semibold uppercase tracking-wide text-neutral-500">Colours</h3>
        {COLOR_NAMES.map(name => (
          <label key={name} className="flex items-center gap-2 text-xs capitalize">
            <input type="color" value={draft.colors[name]} onChange={e => set('colors', { [name]: e.target.value })} aria-label={`${name} colour`} />
            {name}
          </label>
        ))}
      </section>
      <section className="flex flex-col gap-2" aria-label="Typography">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-neutral-500">Typography</h3>
        <ToggleField label="Use Google Fonts (loads fonts from Google)" value={google}
          onChange={v => set('typography', v ? { source: 'google' } : { source: 'system', heading_font: 'system-sans', body_font: 'system-sans' })} />
        {google ? (
          <>
            <TextField label="Heading font (Google Fonts name)" value={draft.typography.heading_font} maxLength={64} onChange={v => set('typography', { heading_font: v })} />
            <TextField label="Body font (Google Fonts name)" value={draft.typography.body_font} maxLength={64} onChange={v => set('typography', { body_font: v })} />
          </>
        ) : (
          <>
            <SelectField label="Heading font" value={draft.typography.heading_font} options={SYSTEM_FONTS} onChange={v => set('typography', { heading_font: String(v) })} />
            <SelectField label="Body font" value={draft.typography.body_font} options={SYSTEM_FONTS} onChange={v => set('typography', { body_font: String(v) })} />
          </>
        )}
        <SelectField label="Text size" value={draft.typography.scale} options={[{ value: 'sm', label: 'Small' }, { value: 'md', label: 'Medium' }, { value: 'lg', label: 'Large' }]}
          onChange={v => set('typography', { scale: v as ThemeSettings['typography']['scale'] })} />
      </section>
      <section className="flex flex-col gap-2" aria-label="Layout">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-neutral-500">Layout</h3>
        <NumberField label="Page width (px)" value={draft.layout.max_width} min={960} max={1920} onChange={v => set('layout', { max_width: v })} />
        <SelectField label="Button corners" value={draft.buttons.radius} options={['none', 'sm', 'md', 'lg', 'full'].map(v => ({ value: v, label: v }))}
          onChange={v => set('buttons', { radius: v as ThemeSettings['buttons']['radius'] })} />
        <SelectField label="Products per row" value={draft.product.grid_columns} options={[2, 3, 4, 5, 6].map(v => ({ value: v, label: String(v) }))}
          onChange={v => set('product', { grid_columns: Number(v) })} />
        <SelectField label="Product image shape" value={draft.product.image_ratio} options={['1:1', '4:3', '3:4', '2:3', '16:9'].map(v => ({ value: v, label: v }))}
          onChange={v => set('product', { image_ratio: v as ThemeSettings['product']['image_ratio'] })} />
      </section>
      <button type="button" disabled={!changed || saving} onClick={() => onSave(draft)}
        className="rounded-md bg-blue-700 px-3 py-2 text-sm font-medium text-white disabled:opacity-40">
        {saving ? 'Saving…' : 'Save settings'}
      </button>
    </div>
  )
}
