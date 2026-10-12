import { useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useTenantSettings } from './hooks/useTenantSettings'
import { apiErrorText } from '../api/apiErrorText'

// Tunes the FP-Growth cross-sell mining GetCartRecommendationsUseCase runs
// for every storefront (e.g. the WooCommerce plugin) request -- lives next
// to ApiKeysCard since both are about that same storefront-facing
// recommendations feature. All three are quality levers, not speed ones
// (Apriori and FP-Growth mine identical itemsets for the same minSupport):
// they decide what counts as a real cross-sell pattern versus coincidence,
// not how fast the mining runs.
export function RecommendationsSettingsCard() {
  const { token } = useAuth()
  const { t } = useTranslation(['settings', 'common'])
  const { settings, save } = useTenantSettings(token ?? '')

  const [confidencePercent, setConfidencePercent] = useState('')
  const [minSupport, setMinSupport] = useState('')
  const [lookbackDays, setLookbackDays] = useState('')
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [saved, setSaved] = useState(false)

  if (!hydrated && settings) {
    setConfidencePercent(settings.bundle_confidence_threshold ? (settings.bundle_confidence_threshold * 100).toString() : '')
    setMinSupport(settings.recommendations_min_support ? settings.recommendations_min_support.toString() : '')
    setLookbackDays(settings.recommendations_lookback_days ? settings.recommendations_lookback_days.toString() : '')
    setHydrated(true)
  }

  async function handleSave() {
    setSaveError('')
    setSaved(false)
    const confidence = confidencePercent.trim() === '' ? 0 : parseFloat(confidencePercent) / 100
    const support = minSupport.trim() === '' ? 0 : Math.round(parseFloat(minSupport))
    const days = lookbackDays.trim() === '' ? 0 : Math.round(parseFloat(lookbackDays))
    if (confidence < 0 || confidence > 1) {
      setSaveError(t('recommendations.confidenceRange'))
      return
    }
    if (support < 0) {
      setSaveError(t('recommendations.supportNegative'))
      return
    }
    if (days < 0) {
      setSaveError(t('recommendations.lookbackNegative'))
      return
    }
    setSaving(true)
    try {
      await save({
        bundle_confidence_threshold: confidence,
        recommendations_min_support: support,
        recommendations_lookback_days: days,
      })
      setSaved(true)
    } catch (err) {
      setSaveError(apiErrorText(err) ?? t('recommendations.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <h3 className="text-sm font-semibold text-ink">{t('recommendations.title')}</h3>
      <p className="text-2xs text-ink-4">
        {t('recommendations.help')}
      </p>

      <div className="grid grid-cols-3 gap-3">
        <div>
          <label htmlFor="bundle-confidence" className="block text-2xs text-ink-4 caps-label mb-1.5">
            {t('recommendations.confidence')}
          </label>
          <input
            id="bundle-confidence"
            type="number" min={0} max={100} step={1}
            value={confidencePercent}
            onChange={e => { setConfidencePercent(e.target.value); setSaved(false) }}
            placeholder="30"
            className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div>
          <label htmlFor="min-support" className="block text-2xs text-ink-4 caps-label mb-1.5">
            {t('recommendations.support')}
          </label>
          <input
            id="min-support"
            type="number" min={0} step={1}
            value={minSupport}
            onChange={e => { setMinSupport(e.target.value); setSaved(false) }}
            placeholder="2"
            className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div>
          <label htmlFor="lookback-days" className="block text-2xs text-ink-4 caps-label mb-1.5">
            {t('recommendations.lookback')}
          </label>
          <input
            id="lookback-days"
            type="number" min={0} step={1}
            value={lookbackDays}
            onChange={e => { setLookbackDays(e.target.value); setSaved(false) }}
            placeholder="180"
            className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
      </div>

      <ul className="text-2xs text-ink-4 space-y-0.5 list-disc list-inside">
        <li><Trans t={t} i18nKey="recommendations.confidenceHelp" components={{ strong: <span className="font-medium" /> }} /></li>
        <li><Trans t={t} i18nKey="recommendations.supportHelp" components={{ strong: <span className="font-medium" /> }} /></li>
        <li><Trans t={t} i18nKey="recommendations.lookbackHelp" components={{ strong: <span className="font-medium" /> }} /></li>
      </ul>

      {saveError && <p className="text-xs text-danger">{saveError}</p>}
      {saved && <p className="text-xs text-success">{t('savedMessage')}</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
      >
        {saving ? t('common:actions.saving') : t('recommendations.save')}
      </button>
    </div>
  )
}
