import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useTenantSettings } from './hooks/useTenantSettings'
import { apiErrorText } from '../api/apiErrorText'
import { DEFAULT_LOW_STOCK_THRESHOLD } from './types'

// The tenant-wide low stock line the inventory list's "Low stock" filter
// uses; a product can override it on its own page. Saves only its own
// field (partial PATCH).
export function LowStockThresholdSection() {
  const { token } = useAuth()
  const { t } = useTranslation(['settings', 'common'])
  const { settings, save } = useTenantSettings(token ?? '')

  const [threshold, setThreshold] = useState(DEFAULT_LOW_STOCK_THRESHOLD)
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [saved, setSaved] = useState(false)

  if (!hydrated && settings) {
    setThreshold(settings.low_stock_threshold ?? DEFAULT_LOW_STOCK_THRESHOLD)
    setHydrated(true)
  }

  async function handleSave() {
    setSaving(true)
    setSaveError('')
    setSaved(false)
    try {
      await save({ low_stock_threshold: threshold })
      setSaved(true)
    } catch (err) {
      setSaveError(apiErrorText(err) ?? t('lowStock.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <div>
        <h3 className="text-sm font-semibold text-ink">{t('lowStock.title')}</h3>
        <p className="text-2xs text-ink-4 mt-0.5">{t('lowStock.help')}</p>
      </div>
      <div className="max-w-[220px]">
        <label htmlFor="low-stock-threshold" className="block text-2xs text-ink-4 caps-label mb-1.5">
          {t('lowStock.threshold')}
        </label>
        <input
          id="low-stock-threshold"
          type="number"
          min={0}
          value={threshold}
          onChange={e => { setThreshold(Math.max(0, parseInt(e.target.value, 10) || 0)); setSaved(false) }}
          className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
      </div>
      {saveError && <p role="alert" className="text-xs text-danger">{saveError}</p>}
      {saved && <p className="text-xs text-success">{t('savedMessage')}</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
      >
        {saving ? t('common:actions.saving') : t('lowStock.save')}
      </button>
    </div>
  )
}
