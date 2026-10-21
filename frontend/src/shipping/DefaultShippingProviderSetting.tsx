import { useState } from 'react'
import { Truck } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useShippingConnections } from './useShippingConnections'
import { shippingConnectionLabel } from './shippingProviders'
import { apiErrorText } from '../api/apiErrorText'

// The select's value for "no explicit pick" -- the oldest connection is
// used then (the backend's null).
const AUTOMATIC = ''

interface DefaultShippingProviderSettingProps {
  token: string
  // Owners/admins change it; everyone else sees it read-only (the backend
  // enforces this too).
  canEdit: boolean
}

// "Default shipping provider": which shipping connection packing
// preselects for a new label. Saves on change and sends only this one
// field.
export function DefaultShippingProviderSetting({ token, canEdit }: DefaultShippingProviderSettingProps) {
  const { t } = useTranslation('shipping')
  const { connections, configuredConnectionId, loading, error: loadError, saveDefault } = useShippingConnections(token)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  if (loading || (connections.length === 0 && !loadError)) return null

  async function handleChange(value: string) {
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      await saveDefault(value === AUTOMATIC ? null : value)
      setSaved(true)
    } catch (err) {
      setError(apiErrorText(err) ?? t('defaultProvider.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  const oldest = connections[0]
  return (
    <section className="flex items-start gap-3 px-4 py-3 rounded-xl bg-raised border border-line">
      <div className="w-8 h-8 rounded-lg bg-accent/10 flex items-center justify-center shrink-0">
        <Truck size={15} className="text-accent" />
      </div>
      <div className="flex-1 min-w-0">
        <label htmlFor="default-shipping-provider" className="block text-xs font-semibold text-ink">{t('defaultProvider.title')}</label>
        <p className="text-2xs text-ink-4 mb-2">{t('defaultProvider.body')}</p>
        {loadError && <p className="text-xs text-danger">{loadError}</p>}
        {connections.length > 0 && (
          <select
            id="default-shipping-provider"
            value={configuredConnectionId ?? AUTOMATIC}
            onChange={e => handleChange(e.target.value)}
            disabled={!canEdit || saving}
            className="w-full max-w-xs px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink disabled:opacity-60"
          >
            <option value={AUTOMATIC}>
              {t('defaultProvider.automatic', { provider: oldest ? shippingConnectionLabel(oldest, connections) : '' })}
            </option>
            {connections.map(c => <option key={c.id} value={c.id}>{shippingConnectionLabel(c, connections)}</option>)}
          </select>
        )}
        {!canEdit && <p className="text-2xs text-ink-4 mt-1">{t('defaultProvider.readOnly')}</p>}
        {error && <p role="alert" className="text-xs text-danger mt-1">{error}</p>}
        {saved && <p role="status" className="text-xs text-success mt-1">{t('defaultProvider.saved')}</p>}
      </div>
    </section>
  )
}
