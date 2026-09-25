import { useState } from 'react'
import { useAuth } from '../auth/AuthContext'
import { useTenantSettings } from './hooks/useTenantSettings'
import { toInput } from './tenantSettingsInput'

// Tunes the FP-Growth cross-sell mining GetCartRecommendationsUseCase runs
// for every storefront (e.g. the WooCommerce plugin) request -- lives next
// to ApiKeysCard since both are about that same storefront-facing
// recommendations feature. All three are quality levers, not speed ones
// (Apriori and FP-Growth mine identical itemsets for the same minSupport):
// they decide what counts as a real cross-sell pattern versus coincidence,
// not how fast the mining runs.
export function RecommendationsSettingsCard() {
  const { token } = useAuth()
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
      setSaveError('Bundle confidence must be between 0 and 100%.')
      return
    }
    if (support < 0) {
      setSaveError('Minimum support cannot be negative.')
      return
    }
    if (days < 0) {
      setSaveError('Lookback window cannot be negative.')
      return
    }
    setSaving(true)
    try {
      await save({
        ...toInput(settings),
        bundle_confidence_threshold: confidence,
        recommendations_min_support: support,
        recommendations_lookback_days: days,
      })
      setSaved(true)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save recommendation settings')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <h3 className="text-sm font-semibold text-ink">Cross-sell recommendations</h3>
      <p className="text-2xs text-ink-4">
        Tunes the pattern-mining behind "You may also like" and "Frequently bought together" on connected storefronts. Leave blank to use CommerceOS' defaults.
      </p>

      <div className="grid grid-cols-3 gap-3">
        <div>
          <label htmlFor="bundle-confidence" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
            Bundle confidence (%)
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
          <label htmlFor="min-support" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
            Minimum support (orders)
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
          <label htmlFor="lookback-days" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
            Lookback window (days)
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
        <li><span className="font-medium">Bundle confidence</span> — of orders containing a cart item, the % that must also contain a candidate before it's flagged bundle-worthy. Higher = fewer, more reliable bundle suggestions.</li>
        <li><span className="font-medium">Minimum support</span> — orders a product pairing must appear in before it counts as a real pattern rather than coincidence. Raise this on a high-volume store to cut noise; keep it low on a small catalog or recommendations may disappear entirely.</li>
        <li><span className="font-medium">Lookback window</span> — how far back order history is mined. Shorter favors recent behavior over stale, since-discontinued patterns; longer gives more data for a seasonal or slow-moving catalog.</li>
      </ul>

      {saveError && <p className="text-xs text-danger">{saveError}</p>}
      {saved && <p className="text-xs text-success">Saved.</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
      >
        {saving ? 'Saving…' : 'Save recommendation settings'}
      </button>
    </div>
  )
}
