import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useTenantSettings } from './hooks/useTenantSettings'
import { apiErrorText } from '../api/apiErrorText'

export function InventorySyncRulesSection() {
  const { token } = useAuth()
  const { t } = useTranslation(['settings', 'common'])
  const { settings, save } = useTenantSettings(token ?? '')

  const [enabled, setEnabled] = useState(false)
  const [thresholdDays, setThresholdDays] = useState(0)
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [saved, setSaved] = useState(false)

  if (!hydrated && settings) {
    setEnabled(settings.inventory_sync_rules_enabled)
    setThresholdDays(settings.batch_expiry_threshold_days)
    setHydrated(true)
  }

  async function handleSave() {
    setSaving(true)
    setSaveError('')
    setSaved(false)
    try {
      await save({ inventory_sync_rules_enabled: enabled, batch_expiry_threshold_days: thresholdDays })
      setSaved(true)
    } catch (err) {
      setSaveError(apiErrorText(err) ?? t('syncRules.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <h3 className="text-sm font-semibold text-ink">{t('syncRules.title')}</h3>
      <div className="flex items-center justify-between">
        <div>
          <span className="text-xs text-ink-2 block">{t('syncRules.enable')}</span>
          <span className="text-2xs text-ink-4">
            {t('syncRules.help')}
          </span>
        </div>
        <label className="relative flex items-center shrink-0 cursor-pointer">
          <input
            type="checkbox"
            role="switch"
            aria-label={t('syncRules.enable')}
            checked={enabled}
            onChange={e => { setEnabled(e.target.checked); setSaved(false) }}
            className="sr-only peer"
          />
          <div className="w-8 h-4 bg-line rounded-full peer peer-checked:bg-accent-solid transition-colors" />
          <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
        </label>
      </div>
      <div className="max-w-[220px]">
        <label htmlFor="batch-expiry-threshold-days" className="block text-2xs text-ink-4 caps-label mb-1.5">
          {t('syncRules.autoExpire')}
        </label>
        <input
          id="batch-expiry-threshold-days"
          type="number"
          min={0}
          value={thresholdDays}
          onChange={e => { setThresholdDays(parseInt(e.target.value, 10) || 0); setSaved(false) }}
          className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
      </div>
      {saveError && <p className="text-xs text-danger">{saveError}</p>}
      {saved && <p className="text-xs text-success">{t('savedMessage')}</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
      >
        {saving ? t('common:actions.saving') : t('syncRules.save')}
      </button>
    </div>
  )
}
