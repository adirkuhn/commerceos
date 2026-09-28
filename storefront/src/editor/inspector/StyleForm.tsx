import type { Breakpoint, NodeStyle, StyleProperty } from '../../core/document'
import type { StyleGroup } from '../../components/types'
import { Field, SelectField } from './fields'
import { AssetPicker } from './pickers'
import { controlsFor, setStyleValue, type StyleControl } from './styleControls'

const COLOR_TOKENS = ['primary', 'secondary', 'accent', 'text', 'muted', 'background', 'surface', 'transparent']
const HEX = /^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/

const BP_LABELS: Record<Breakpoint, string> = { base: 'all screens', tablet: 'tablet and mobile', mobile: 'mobile' }

function ColorControl({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  const custom = HEX.test(value)
  return (
    <Field label={label}>
      {id => (
        <div className="flex gap-2">
          <select id={id} className="flex-1 rounded-md border border-neutral-300 px-2 py-1.5 text-sm" value={custom ? 'custom' : value}
            onChange={e => onChange(e.target.value === 'custom' ? '#000000' : e.target.value)}>
            <option value="">Default</option>
            {COLOR_TOKENS.map(t => <option key={t} value={t}>{t === 'transparent' ? 'None' : `Theme ${t}`}</option>)}
            <option value="custom">Custom…</option>
          </select>
          {custom && <input type="color" aria-label={`${label} custom colour`} value={value.length === 4 ? `#${[...value.slice(1)].map(c => c + c).join('')}` : value}
            onChange={e => onChange(e.target.value)} className="h-8 w-10" />}
        </div>
      )}
    </Field>
  )
}

function Control({ control, value, inherited, onChange }: { control: StyleControl; value: string; inherited?: string; onChange: (v: string) => void }) {
  if (control.kind === 'color') return <ColorControl label={control.label} value={value} onChange={onChange} />
  if (control.property === 'background_image') return <AssetPicker label={control.label} value={value} onChange={onChange} />
  return (
    <SelectField label={control.label} value={value} options={control.options ?? []} allowEmpty={inherited ? `Same as larger screens (${inherited})` : 'Default'}
      onChange={v => onChange(String(v))} />
  )
}

/** Style controls for one breakpoint; smaller screens inherit what they don't override. */
export function StyleForm({ groups, style, breakpoint, onChange }: {
  groups: StyleGroup[]; style: NodeStyle; breakpoint: Breakpoint; onChange: (next: NodeStyle) => void
}) {
  const inheritedValue = (p: StyleProperty) =>
    breakpoint === 'mobile' ? style.tablet?.[p] ?? style.base?.[p] : breakpoint === 'tablet' ? style.base?.[p] : undefined
  return (
    <div className="flex flex-col gap-4">
      <p className="text-xs text-neutral-500">Editing styles for {BP_LABELS[breakpoint]}. Switch the preview size to style other screens.</p>
      {controlsFor(groups).map(g => (
        <fieldset key={g.group} className="flex flex-col gap-2">
          <legend className="mb-1 text-xs font-semibold uppercase tracking-wide text-neutral-500">{g.label}</legend>
          {g.controls.map(c => (
            <Control key={c.property} control={c} value={style[breakpoint]?.[c.property] ?? ''} inherited={inheritedValue(c.property)}
              onChange={v => onChange(setStyleValue(style, breakpoint, c.property, v))} />
          ))}
        </fieldset>
      ))}
    </div>
  )
}
