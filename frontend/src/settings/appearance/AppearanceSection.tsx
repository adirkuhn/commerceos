import { useEffect, useRef, useState } from 'react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, RotateCcw } from 'lucide-react'
import { useAuth } from '../../auth/AuthContext'
import { apiErrorText } from '../../api/apiErrorText'
import { useAppearance } from '../../theme/AppearanceProvider'
import {
  APPEARANCE_CORNERS, APPEARANCE_DENSITIES, APPEARANCE_FONTS, APPEARANCE_TEXT_SIZES, DEFAULT_APPEARANCE, FONT_SPECS,
  darkenToContrast, fontStylesheetHref, meetsAccentContrast, type AppearanceSettings,
} from '../../theme/appearance'
import { ColorField, Segmented, isHexColor } from './appearanceFields'

// Same roles as the backend's RequireRoleForWrites on the settings routes.
const EDITOR_ROLES = ['customer_owner', 'customer_admin', 'super_admin']

function settingsOf(a: AppearanceSettings): AppearanceSettings {
  return {
    font: a.font, accent_color: a.accent_color, sidebar_color: a.sidebar_color,
    corners: a.corners, text_size: a.text_size, density: a.density,
  }
}

// The font cards are set in their own typeface, so those stylesheets load
// while this section is open.
function useFontSamples() {
  useEffect(() => {
    const links = APPEARANCE_FONTS.map(fontStylesheetHref).filter((h): h is string => h !== null).map(href => {
      const link = document.createElement('link')
      link.rel = 'stylesheet'
      link.href = href
      link.dataset.appearanceSample = ''
      document.head.appendChild(link)
      return link
    })
    return () => links.forEach(l => l.remove())
  }, [])
}

// Settings > Appearance: how the admin looks for everyone in this shop
// (both looks). Every change previews live on the whole app; leaving
// without saving puts the saved look back.
export function AppearanceSection() {
  const { t } = useTranslation(['settings', 'common'])
  const { user } = useAuth()
  const { appearance, error: loadError, preview, save } = useAppearance()
  const canEdit = EDITOR_ROLES.includes(user?.role ?? '')
  useFontSamples()

  const saved = settingsOf(appearance ?? DEFAULT_APPEARANCE)
  const [edits, setEdits] = useState<AppearanceSettings | null>(null)
  const draft = edits ?? saved
  const [saving, setSaving] = useState(false)
  const [status, setStatus] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const colorsValid = isHexColor(draft.accent_color) && isHexColor(draft.sidebar_color)
  const accentReadable = isHexColor(draft.accent_color) && meetsAccentContrast(draft.accent_color)
  const dirty = edits !== null && JSON.stringify(edits) !== JSON.stringify(saved)

  // Leaving the section shows the saved look again.
  const previewRef = useRef(preview)
  previewRef.current = preview
  useEffect(() => () => previewRef.current(null), [])

  function change(next: AppearanceSettings) {
    setEdits(next)
    setStatus(null)
    if (isHexColor(next.accent_color) && isHexColor(next.sidebar_color)) preview(next)
  }
  const set = <K extends keyof AppearanceSettings>(key: K) => (value: AppearanceSettings[K]) => change({ ...draft, [key]: value })

  async function submit() {
    setSaving(true)
    setStatus(null)
    try {
      await save(draft)
      setEdits(null)
      setStatus({ kind: 'ok', text: t('common:status.saved') })
    } catch (e) {
      setStatus({ kind: 'error', text: apiErrorText(e) ?? t('appearance.saveFailed') })
    } finally {
      setSaving(false)
    }
  }

  const suggestion = isHexColor(draft.accent_color) && !accentReadable ? darkenToContrast(draft.accent_color) : null

  return (
    <div className="max-w-2xl space-y-8">
      <div>
        <h2 className="text-lg font-semibold text-ink">{t('appearance.title')}</h2>
        <p className="mt-1 text-sm text-ink-3">{t('appearance.intro')}</p>
        {loadError && <p role="alert" className="mt-2 text-sm text-danger">{loadError}</p>}
        {!canEdit && <p className="mt-2 text-xs text-ink-3">{t('appearance.readOnly')}</p>}
      </div>

      <fieldset disabled={!canEdit}>
        <legend className="text-sm font-medium text-ink mb-2">{t('appearance.font')}</legend>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
          {APPEARANCE_FONTS.map(font => {
            const spec = FONT_SPECS[font]
            const checked = draft.font === font
            return (
              <label
                key={font}
                className={clsx(
                  'flex items-start gap-3 rounded-lg border px-4 py-3 cursor-pointer has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-focus/60',
                  checked ? 'border-accent bg-accent/5' : 'border-line bg-raised hover:border-ink-4',
                  !canEdit && 'cursor-default',
                )}
              >
                <input type="radio" name="appearance-font" aria-label={spec.label} checked={checked} onChange={() => set('font')(font)} className="mt-1 accent-[rgb(var(--accent))]" />
                <span style={{ fontFamily: `${spec.sans}, system-ui, sans-serif` }}>
                  <span className="block text-base text-ink">{spec.label}</span>
                  <span className="block text-sm text-ink-3" aria-hidden>{t('appearance.fontSample')}</span>
                </span>
              </label>
            )
          })}
        </div>
      </fieldset>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
        <ColorField label={t('appearance.accent')} value={draft.accent_color} disabled={!canEdit} onChange={set('accent_color')}>
          <p className="mt-1 text-xs text-ink-4">{t('appearance.accentHint')}</p>
          {suggestion && (
            <div className="mt-2 flex items-start gap-2 rounded-lg bg-warning/10 px-3 py-2 text-xs text-warning">
              <AlertTriangle size={14} className="mt-0.5 shrink-0" aria-hidden />
              <span className="flex-1">{t('appearance.lowContrast')}</span>
              <button type="button" onClick={() => set('accent_color')(suggestion)} className="shrink-0 inline-flex items-center gap-1.5 font-medium underline">
                <span className="inline-block h-3 w-3 rounded-sm" style={{ background: suggestion }} aria-hidden />
                {t('appearance.useSuggestion', { color: suggestion })}
              </button>
            </div>
          )}
        </ColorField>
        <ColorField label={t('appearance.sidebar')} value={draft.sidebar_color} disabled={!canEdit} onChange={set('sidebar_color')}>
          <p className="mt-1 text-xs text-ink-4">{t('appearance.sidebarHint')}</p>
        </ColorField>
      </div>

      <div className="flex flex-wrap gap-x-10 gap-y-6">
        <Segmented name="appearance-corners" legend={t('appearance.corners')} value={draft.corners} disabled={!canEdit} onChange={set('corners')}
          options={APPEARANCE_CORNERS.map(v => ({ value: v, label: t(`appearance.cornersOption.${v}`) }))} />
        <Segmented name="appearance-text-size" legend={t('appearance.textSize')} value={draft.text_size} disabled={!canEdit} onChange={set('text_size')}
          options={APPEARANCE_TEXT_SIZES.map(v => ({ value: v, label: t(`appearance.textSizeOption.${v}`) }))} />
        <Segmented name="appearance-density" legend={t('appearance.density')} value={draft.density} disabled={!canEdit} onChange={set('density')}
          options={APPEARANCE_DENSITIES.map(v => ({ value: v, label: t(`appearance.densityOption.${v}`) }))} />
      </div>

      {canEdit && (
        <div className="flex flex-wrap items-center gap-3 border-t border-line pt-5">
          <button
            type="button"
            onClick={submit}
            disabled={saving || !dirty || !colorsValid || !accentReadable}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-sm font-semibold hover:bg-accent-solid-hover disabled:opacity-50"
          >
            {saving ? t('common:actions.saving') : t('appearance.save')}
          </button>
          <button
            type="button"
            onClick={() => change(DEFAULT_APPEARANCE)}
            className="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium text-ink-3 hover:bg-hover"
          >
            <RotateCcw size={14} aria-hidden /> {t('appearance.resetDefaults')}
          </button>
          {status && (
            <span role={status.kind === 'error' ? 'alert' : 'status'} className={clsx('text-sm', status.kind === 'error' ? 'text-danger' : 'text-success')}>
              {status.text}
            </span>
          )}
        </div>
      )}
    </div>
  )
}
