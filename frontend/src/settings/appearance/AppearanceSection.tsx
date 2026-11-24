import { useEffect, useRef, useState } from 'react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { RotateCcw } from 'lucide-react'
import { useAuth } from '../../auth/AuthContext'
import { apiErrorText } from '../../api/apiErrorText'
import { useAppearance } from '../../theme/AppearanceProvider'
import { LOOKS } from '../../theme/lookPreference'
import {
  APPEARANCE_CORNERS, APPEARANCE_DENSITIES, APPEARANCE_TEXT_SIZES, DEFAULT_APPEARANCE, type AppearanceSettings,
} from '../../theme/appearance'
import { LookCards, Segmented } from './appearanceFields'

// Same roles as the backend's RequireRoleForWrites on the settings routes.
const EDITOR_ROLES = ['customer_owner', 'customer_admin', 'super_admin']

function settingsOf(a: AppearanceSettings): AppearanceSettings {
  return { default_look: a.default_look, corners: a.corners, text_size: a.text_size, density: a.density }
}

// Settings > Appearance: the shop's default look and how the admin is laid
// out for everyone in this shop. Every change previews live on the whole
// app; leaving without saving puts the saved appearance back.
export function AppearanceSection() {
  const { t } = useTranslation(['settings', 'common'])
  const { user } = useAuth()
  const { appearance, error: loadError, preview, save } = useAppearance()
  const canEdit = EDITOR_ROLES.includes(user?.role ?? '')

  const saved = settingsOf(appearance ?? DEFAULT_APPEARANCE)
  const [edits, setEdits] = useState<AppearanceSettings | null>(null)
  const draft = edits ?? saved
  const [saving, setSaving] = useState(false)
  const [status, setStatus] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)
  const dirty = edits !== null && JSON.stringify(edits) !== JSON.stringify(saved)

  // Leaving the section shows the saved appearance again.
  const previewRef = useRef(preview)
  previewRef.current = preview
  useEffect(() => () => previewRef.current(null), [])

  function change(next: AppearanceSettings) {
    setEdits(next)
    setStatus(null)
    preview(next)
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

  return (
    <div className="max-w-2xl space-y-8">
      <div>
        <h2 className="text-lg font-semibold text-ink">{t('appearance.title')}</h2>
        <p className="mt-1 text-sm text-ink-3">{t('appearance.intro')}</p>
        {loadError && <p role="alert" className="mt-2 text-sm text-danger">{loadError}</p>}
        {!canEdit && <p className="mt-2 text-xs text-ink-3">{t('appearance.readOnly')}</p>}
      </div>

      <div>
        <LookCards
          label={t('appearance.defaultLook')}
          value={draft.default_look}
          disabled={!canEdit}
          onChange={set('default_look')}
          options={LOOKS.map(look => ({
            value: look,
            name: t(`appearance.lookOption.${look}.name`),
            description: t(`appearance.lookOption.${look}.description`),
          }))}
        />
        <p className="mt-2 text-xs text-ink-4">{t('appearance.lookNote')}</p>
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
            disabled={saving || !dirty}
            className="px-4 py-2 rounded-md bg-accent-solid text-white text-sm font-semibold hover:bg-accent-solid-hover disabled:opacity-50"
          >
            {saving ? t('common:actions.saving') : t('appearance.save')}
          </button>
          <button
            type="button"
            onClick={() => change(DEFAULT_APPEARANCE)}
            className="inline-flex items-center gap-1.5 px-3 py-2 rounded-md text-sm font-medium text-ink-3 hover:bg-hover"
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
