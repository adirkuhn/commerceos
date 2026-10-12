import { useTranslation } from 'react-i18next'
import type { Breakpoint, NodeStyle, StyleProperty } from '../../core/document'
import { useEditorLabels } from '../i18n/useEditorLabels'
import type { StyleGroup } from '../../components/types'
import { Field, SelectField } from './fields'
import { AssetPicker } from './pickers'
import { controlsFor, setStyleValue, type StyleControl } from './styleControls'

const COLOR_TOKENS = ['primary', 'secondary', 'accent', 'text', 'muted', 'background', 'surface', 'transparent']
const HEX = /^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/


function ColorControl({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  const custom = HEX.test(value)
  const { t } = useTranslation('editor')
  const themeColor = (c: string) => t(`settings.colors.${c}` as 'settings.colors.primary')
  return (
    <Field label={label}>
      {id => (
        <div className="flex gap-2">
          <select id={id} className="flex-1 rounded-ed border border-ed-line bg-ed-sunken px-2.5 py-1.5 text-sm text-ed-ink focus:outline-none focus:ring-2 focus:ring-ed-accent/60" value={custom ? 'custom' : value}
            onChange={e => onChange(e.target.value === 'custom' ? '#000000' : e.target.value)}>
            <option value="">{t('styleForm.default')}</option>
            {COLOR_TOKENS.map(c => <option key={c} value={c}>{c === 'transparent' ? t('styleForm.colorNone') : t('styleForm.colorTheme', { name: themeColor(c) })}</option>)}
            <option value="custom">{t('styleForm.custom')}</option>
          </select>
          {custom && <input type="color" className="h-8 w-10 shrink-0 cursor-pointer rounded-ed border border-ed-line bg-ed-surface p-0.5" aria-label={t('styleForm.customColour', { label })} value={value.length === 4 ? `#${[...value.slice(1)].map(c => c + c).join('')}` : value}
            onChange={e => onChange(e.target.value)} />}
        </div>
      )}
    </Field>
  )
}

function Control({ control, value, inherited, onChange }: { control: StyleControl; value: string; inherited?: string; onChange: (v: string) => void }) {
  const { t } = useTranslation('editor')
  const labels = useEditorLabels()
  const label = labels.styleProperty(control)
  if (control.kind === 'color') return <ColorControl label={label} value={value} onChange={onChange} />
  if (control.property === 'background_image') return <AssetPicker label={label} value={value} onChange={onChange} />
  const options = (control.options ?? []).map(o => ({ value: o.value, label: labels.styleOption(control, o) }))
  const inheritedLabel = inherited ? options.find(o => o.value === inherited)?.label ?? inherited : ''
  return (
    <SelectField label={label} value={value} options={options} allowEmpty={inherited ? t('styleForm.sameAsLarger', { value: inheritedLabel }) : t('styleForm.default')}
      onChange={v => onChange(String(v))} />
  )
}

/** Style controls for one breakpoint; smaller screens inherit what they don't override. */
export function StyleForm({ groups, style, breakpoint, onChange }: {
  groups: StyleGroup[]; style: NodeStyle; breakpoint: Breakpoint; onChange: (next: NodeStyle) => void
}) {
  const { t } = useTranslation('editor')
  const labels = useEditorLabels()
  const screens: Record<Breakpoint, string> = { base: t('styleForm.screens.base'), tablet: t('styleForm.screens.tablet'), mobile: t('styleForm.screens.mobile') }
  const inheritedValue = (p: StyleProperty) =>
    breakpoint === 'mobile' ? style.tablet?.[p] ?? style.base?.[p] : breakpoint === 'tablet' ? style.base?.[p] : undefined
  return (
    <div className="flex flex-col gap-4">
      <p className="text-xs text-ed-ink-4">{t('styleForm.editingFor', { screens: screens[breakpoint] })}</p>
      {controlsFor(groups).map(g => (
        <fieldset key={g.group} className="flex flex-col gap-2">
          <legend className="mb-1 text-sm font-semibold text-ed-ink">{labels.styleGroup(g.group, g.label)}</legend>
          {g.controls.map(c => (
            <Control key={c.property} control={c} value={style[breakpoint]?.[c.property] ?? ''} inherited={inheritedValue(c.property)}
              onChange={v => onChange(setStyleValue(style, breakpoint, c.property, v))} />
          ))}
        </fieldset>
      ))}
    </div>
  )
}
