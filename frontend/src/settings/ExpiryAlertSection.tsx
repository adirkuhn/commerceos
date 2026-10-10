import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useTenantSettings } from './hooks/useTenantSettings'
import { toInput } from './tenantSettingsInput'
import { EXPIRY_ALERT_WINDOWS, type ExpiryAlertWindow } from './types'

// Settings > Inventory: the daily stock-close-to-expiry report. The email
// itself is a Message Center template + rule; this only turns the report on
// and picks which "days before expiry" windows it covers.
export function ExpiryAlertSection() {
  const { token } = useAuth()
  const { t } = useTranslation(['settings', 'common'])
  const { settings, save } = useTenantSettings(token ?? '')

  const [enabled, setEnabled] = useState(false)
  const [windows, setWindows] = useState<ExpiryAlertWindow[]>([])
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  if (!hydrated && settings) {
    setEnabled(settings.expiry_alert_enabled ?? false)
    setWindows(settings.expiry_alert_windows ?? [])
    setHydrated(true)
  }

  const missingWindow = enabled && windows.length === 0

  function toggleWindow(w: ExpiryAlertWindow, on: boolean) {
    setWindows(prev => EXPIRY_ALERT_WINDOWS.filter(o => (o === w ? on : prev.includes(o))))
    setSaved(false)
  }

  async function handleSave() {
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      await save({ ...toInput(settings), expiry_alert_enabled: enabled, expiry_alert_windows: windows })
      setSaved(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('expiryAlert.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <div>
        <h3 className="text-sm font-semibold text-ink">{t('expiryAlert.title')}</h3>
        <p className="text-2xs text-ink-4 mt-0.5">{t('expiryAlert.subtitle')}</p>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-xs text-ink-2">{t('expiryAlert.enable')}</span>
        <label className="relative flex items-center shrink-0 cursor-pointer">
          <input
            type="checkbox"
            role="switch"
            aria-label={t('expiryAlert.enable')}
            checked={enabled}
            onChange={e => { setEnabled(e.target.checked); setSaved(false) }}
            className="sr-only peer"
          />
          <div className="w-8 h-4 dark:bg-night-700 bg-gray-300 rounded-full peer peer-checked:bg-accent-solid transition-colors" />
          <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
        </label>
      </div>
      <fieldset className="space-y-1.5">
        <legend className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">{t('expiryAlert.windows')}</legend>
        <div className="flex flex-wrap gap-2">
          {EXPIRY_ALERT_WINDOWS.map(w => (
            <label key={w} className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-line bg-surface text-xs text-ink-2 cursor-pointer has-[:checked]:border-accent">
              <input
                type="checkbox"
                checked={windows.includes(w)}
                onChange={e => toggleWindow(w, e.target.checked)}
                className="accent-accent"
              />
              {t('expiryAlert.days', { count: w })}
            </label>
          ))}
        </div>
        <p className="text-2xs text-ink-4">{t('expiryAlert.windowsHelp')}</p>
      </fieldset>
      {missingWindow && <p className="text-2xs text-warning">{t('expiryAlert.windowRequired')}</p>}
      {error && <p role="alert" className="text-2xs text-danger">{error}</p>}
      {saved && <p className="text-2xs text-success">{t('common:status.saved')}</p>}
      <button
        onClick={handleSave}
        disabled={saving || !settings || missingWindow}
        className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
      >
        {saving ? t('common:actions.saving') : t('expiryAlert.save')}
      </button>
    </div>
  )
}
