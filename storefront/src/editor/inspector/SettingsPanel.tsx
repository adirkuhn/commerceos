import { useEffect, useState } from 'react'
import { THEME_STYLES } from '../../theme/styles'
import { useTranslation } from 'react-i18next'
import type { ThemeSettings } from '../../core/document'
import { NumberField, SelectField, TextField, ToggleField } from './fields'
import { AssetPicker } from './pickers'

const COLOR_NAMES: (keyof ThemeSettings['colors'])[] = ['primary', 'secondary', 'accent', 'text', 'muted', 'background', 'surface']

const assetRef = (id?: string | null) => (id ? `asset:${id}` : '')
const assetId = (ref: string) => ref.replace(/^asset:/, '') || null

/** Theme-wide design settings: brand, colours, fonts, layout, buttons, products. */
export function SettingsPanel({ settings, onSave, saving }: { settings: ThemeSettings; onSave: (s: ThemeSettings) => void; saving: boolean }) {
  const [draft, setDraft] = useState(settings)
  useEffect(() => setDraft(settings), [settings])
  const set = <K extends Exclude<keyof ThemeSettings, 'style'>>(key: K, value: Partial<ThemeSettings[K]>) =>
    setDraft(d => ({ ...d, [key]: { ...d[key], ...value } }))
  const changed = JSON.stringify(draft) !== JSON.stringify(settings)
  const google = draft.typography.source === 'google'
  const { t } = useTranslation('editor')
  const systemFonts = [
    { value: 'system-sans', label: t('settings.fonts.sans') }, { value: 'system-serif', label: t('settings.fonts.serif') },
    { value: 'system-rounded', label: t('settings.fonts.rounded') }, { value: 'system-mono', label: t('settings.fonts.mono') },
  ]
  const colorNames: Record<(typeof COLOR_NAMES)[number], string> = {
    primary: t('settings.colors.primary'), secondary: t('settings.colors.secondary'), accent: t('settings.colors.accent'),
    text: t('settings.colors.text'), muted: t('settings.colors.muted'), background: t('settings.colors.background'), surface: t('settings.colors.surface'),
  }
  const styleNames: Record<NonNullable<ThemeSettings['style']>, string> = { default: t('settings.styles.default'), supermarket: t('settings.styles.supermarket') }
  const corners: Record<ThemeSettings['buttons']['radius'], string> = {
    none: t('settings.corners.none'), sm: t('settings.corners.sm'), md: t('settings.corners.md'), lg: t('settings.corners.lg'), full: t('settings.corners.full'),
  }

  return (
    <div className="flex flex-col gap-5">
      <h2 className="font-semibold">{t('settings.title')}</h2>
      <SelectField label={t('settings.style')} value={draft.style ?? 'default'} options={THEME_STYLES.map(v => ({ value: v, label: styleNames[v] }))}
        onChange={v => setDraft(d => ({ ...d, style: v as ThemeSettings['style'] }))} />
      <section className="flex flex-col gap-2" aria-label={t('settings.brand')}>
        <h3 className="text-xs font-semibold uppercase tracking-wide text-neutral-500">{t('settings.brand')}</h3>
        <AssetPicker label={t('settings.logo')} value={assetRef(draft.brand.logo_asset_id)} onChange={r => set('brand', { logo_asset_id: assetId(r) })} />
        <AssetPicker label={t('settings.favicon')} value={assetRef(draft.brand.favicon_asset_id)} onChange={r => set('brand', { favicon_asset_id: assetId(r) })} />
      </section>
      <section className="grid grid-cols-2 gap-2" aria-label={t('settings.colours')}>
        <h3 className="col-span-2 text-xs font-semibold uppercase tracking-wide text-neutral-500">{t('settings.colours')}</h3>
        {COLOR_NAMES.map(name => (
          <label key={name} className="flex items-center gap-2 text-xs">
            <input type="color" value={draft.colors[name]} onChange={e => set('colors', { [name]: e.target.value })} aria-label={t('settings.colourLabel', { name: colorNames[name] })} />
            {colorNames[name]}
          </label>
        ))}
      </section>
      <section className="flex flex-col gap-2" aria-label={t('settings.typography')}>
        <h3 className="text-xs font-semibold uppercase tracking-wide text-neutral-500">{t('settings.typography')}</h3>
        <ToggleField label={t('settings.useGoogle')} value={google}
          onChange={v => set('typography', v ? { source: 'google' } : { source: 'system', heading_font: 'system-sans', body_font: 'system-sans' })} />
        {google ? (
          <>
            <TextField label={t('settings.headingFontGoogle')} value={draft.typography.heading_font} maxLength={64} onChange={v => set('typography', { heading_font: v })} />
            <TextField label={t('settings.bodyFontGoogle')} value={draft.typography.body_font} maxLength={64} onChange={v => set('typography', { body_font: v })} />
          </>
        ) : (
          <>
            <SelectField label={t('settings.headingFont')} value={draft.typography.heading_font} options={systemFonts} onChange={v => set('typography', { heading_font: String(v) })} />
            <SelectField label={t('settings.bodyFont')} value={draft.typography.body_font} options={systemFonts} onChange={v => set('typography', { body_font: String(v) })} />
          </>
        )}
        <SelectField label={t('settings.textSize')} value={draft.typography.scale} options={[{ value: 'sm', label: t('settings.sizes.sm') }, { value: 'md', label: t('settings.sizes.md') }, { value: 'lg', label: t('settings.sizes.lg') }]}
          onChange={v => set('typography', { scale: v as ThemeSettings['typography']['scale'] })} />
      </section>
      <section className="flex flex-col gap-2" aria-label={t('settings.layout')}>
        <h3 className="text-xs font-semibold uppercase tracking-wide text-neutral-500">{t('settings.layout')}</h3>
        <NumberField label={t('settings.pageWidth')} value={draft.layout.max_width} min={960} max={1920} onChange={v => set('layout', { max_width: v })} />
        <SelectField label={t('settings.buttonCorners')} value={draft.buttons.radius} options={(['none', 'sm', 'md', 'lg', 'full'] as const).map(v => ({ value: v, label: corners[v] }))}
          onChange={v => set('buttons', { radius: v as ThemeSettings['buttons']['radius'] })} />
        <SelectField label={t('settings.productsPerRow')} value={draft.product.grid_columns} options={[2, 3, 4, 5, 6].map(v => ({ value: v, label: String(v) }))}
          onChange={v => set('product', { grid_columns: Number(v) })} />
        <SelectField label={t('settings.imageShape')} value={draft.product.image_ratio} options={['1:1', '4:3', '3:4', '2:3', '16:9'].map(v => ({ value: v, label: v }))}
          onChange={v => set('product', { image_ratio: v as ThemeSettings['product']['image_ratio'] })} />
      </section>
      <button type="button" disabled={!changed || saving} onClick={() => onSave(draft)}
        className="rounded-md bg-blue-700 px-3 py-2 text-sm font-medium text-white disabled:opacity-40">
        {saving ? t('settings.saving') : t('settings.save')}
      </button>
    </div>
  )
}
