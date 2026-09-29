import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useTenantSettings } from './hooks/useTenantSettings'
import { toInput } from './tenantSettingsInput'

export function PricingMarginSection() {
  const { token } = useAuth()
  const { t } = useTranslation(['settings', 'common'])
  const { settings, save } = useTenantSettings(token ?? '')

  const [markupPercent, setMarkupPercent] = useState('')
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [saved, setSaved] = useState(false)

  if (!hydrated && settings) {
    setMarkupPercent(settings.default_target_markup_basis_points ? (settings.default_target_markup_basis_points / 100).toString() : '')
    setHydrated(true)
  }

  async function handleSave() {
    setSaveError('')
    setSaved(false)
    const basisPoints = markupPercent.trim() === '' ? 0 : Math.round(parseFloat(markupPercent) * 100)
    if (basisPoints < 0) {
      setSaveError(t('markup.negative'))
      return
    }
    setSaving(true)
    try {
      await save({ ...toInput(settings), default_target_markup_basis_points: basisPoints })
      setSaved(true)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : t('markup.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <h3 className="text-sm font-semibold text-ink">{t('markup.title')}</h3>
      <div className="max-w-[220px]">
        <label htmlFor="default-target-markup" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
          {t('markup.label')}
        </label>
        <input
          id="default-target-markup"
          type="number" min={0} step={0.01}
          value={markupPercent}
          onChange={e => { setMarkupPercent(e.target.value); setSaved(false) }}
          placeholder={t('markup.placeholder')}
          className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
        <p className="text-2xs text-ink-4 mt-1">
          {t('markup.help')}
        </p>
      </div>
      {saveError && <p className="text-xs text-danger">{saveError}</p>}
      {saved && <p className="text-xs text-success">{t('savedMessage')}</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
      >
        {saving ? t('common:actions.saving') : t('markup.save')}
      </button>
    </div>
  )
}
