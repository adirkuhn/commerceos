import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import * as integrationsApi from '../integrations/api'
import type { SalesChannel } from '../integrations/types'
import * as settingsApi from './api'
import type { ApiKey } from './types'

// Tenant-wide, not tied to any one Connection -- a storefront (e.g. the
// WooCommerce plugin) calls CommerceOS' cart-recommendations API with one
// of these, and that call only needs a tenant, not a specific integration
// connection (a tenant can have several, see the connections list above).
// Several named, independently revocable keys, mirroring
// ProviderWebhooksPanel's list+create+delete shape.
export function ApiKeysCard() {
  const { token } = useAuth()
  const { t } = useTranslation(['settings', 'common'])
  const [keys, setKeys] = useState<ApiKey[]>([])
  const [channels, setChannels] = useState<SalesChannel[]>([])
  const [loading, setLoading] = useState(true)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [salesChannelId, setSalesChannelId] = useState('')
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')
  const [justCreatedKey, setJustCreatedKey] = useState('')
  const [copied, setCopied] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)

  const internalChannels = channels.filter(c => c.type === 'internal')

  useEffect(() => {
    let cancelled = false
    Promise.all([settingsApi.listApiKeys(token ?? ''), integrationsApi.listSalesChannels(token ?? '')])
      .then(([keysResult, channelsResult]) => { if (!cancelled) { setKeys(keysResult); setChannels(channelsResult) } })
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : t('apiKeys.loadFailed')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, t])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setCreating(true)
    try {
      const created = await settingsApi.createApiKey(token ?? '', name.trim(), description.trim(), salesChannelId || null)
      setKeys(prev => [created, ...prev])
      setJustCreatedKey(created.key)
      setName('')
      setDescription('')
      setSalesChannelId('')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('apiKeys.createFailed'))
    } finally {
      setCreating(false)
    }
  }

  function channelName(id: string | null): string {
    if (!id) return t('apiKeys.allChannels')
    return channels.find(c => c.id === id)?.name ?? t('apiKeys.unknownChannel')
  }

  async function handleRevoke(id: string) {
    setBusyId(id)
    setError('')
    try {
      await settingsApi.revokeApiKey(token ?? '', id)
      setKeys(prev => prev.map(k => (k.id === id ? { ...k, revoked_at: new Date().toISOString() } : k)))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('apiKeys.revokeFailed'))
    } finally {
      setBusyId(null)
    }
  }

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(justCreatedKey)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard access can be denied (permissions, insecure context) --
      // the value is already shown selectable on screen, so this is a
      // convenience failure, not a blocker; no error surfaced for it.
    }
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <div>
        <p className="text-xs font-medium text-ink">{t('apiKeys.title')}</p>
        <p className="text-2xs text-ink-4 mt-0.5">
          {t('apiKeys.help')}
        </p>
      </div>

      {justCreatedKey && (
        <div>
          <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
            {t('apiKeys.newKey')}
          </label>
          <div className="flex items-center gap-2">
            <code className="flex-1 px-3 py-2 text-2xs font-mono rounded-lg bg-sunken border border-line text-ink-2 overflow-x-auto whitespace-nowrap">
              {justCreatedKey}
            </code>
            <button
              type="button"
              onClick={handleCopy}
              className="px-2.5 py-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors shrink-0"
            >
              {copied ? t('apiKeys.copied') : t('apiKeys.copy')}
            </button>
          </div>
          <p className="mt-1.5 text-2xs text-warning">
            {t('apiKeys.shownOnce')}
          </p>
        </div>
      )}

      {error && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error}</p>}

      <form onSubmit={handleCreate} className="flex items-end gap-2">
        <div className="flex-1">
          <label htmlFor="api-key-name" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
            {t('common:fields.name')}
          </label>
          <input
            id="api-key-name" required value={name} onChange={e => setName(e.target.value)}
            placeholder={t('apiKeys.namePlaceholder')}
            className="w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink"
          />
        </div>
        <div className="flex-1">
          <label htmlFor="api-key-description" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
            {t('common:fields.description')}
          </label>
          <input
            id="api-key-description" value={description} onChange={e => setDescription(e.target.value)}
            placeholder={t('apiKeys.optional')}
            className="w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink"
          />
        </div>
        {internalChannels.length > 0 && (
          <div className="flex-1">
            <label htmlFor="api-key-channel" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              {t('apiKeys.salesChannel')}
            </label>
            <select
              id="api-key-channel" value={salesChannelId} onChange={e => setSalesChannelId(e.target.value)}
              className="w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink"
            >
              <option value="">{t('apiKeys.allChannelsTenant')}</option>
              {internalChannels.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
            </select>
          </div>
        )}
        <button
          type="submit" disabled={creating}
          className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
        >
          {creating ? t('apiKeys.creating') : t('apiKeys.create')}
        </button>
      </form>

      {loading ? (
        <p className="text-xs text-ink-4">{t('common:status.loading')}</p>
      ) : keys.length === 0 ? (
        <p className="text-xs text-ink-4">{t('apiKeys.none')}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-left text-ink-4 border-b border-line">
                <th className="py-2 pr-3 font-medium">{t('common:fields.name')}</th>
                <th className="py-2 pr-3 font-medium">{t('common:fields.description')}</th>
                <th className="py-2 pr-3 font-medium">{t('apiKeys.scope')}</th>
                <th className="py-2 pr-3 font-medium">{t('common:fields.status')}</th>
                <th className="py-2 pr-3 font-medium"></th>
              </tr>
            </thead>
            <tbody>
              {keys.map(k => (
                <tr key={k.id} className="border-b border-line last:border-0">
                  <td className="py-2 pr-3 text-ink-2">{k.name}</td>
                  <td className="py-2 pr-3 text-ink-3 truncate max-w-[220px]">{k.description}</td>
                  <td className="py-2 pr-3 text-ink-3">{channelName(k.sales_channel_id)}</td>
                  <td className="py-2 pr-3">
                    {k.revoked_at ? (
                      <span className="text-2xs text-ink-4">{t('apiKeys.revoked')}</span>
                    ) : (
                      <span className="text-2xs text-success bg-green-500/15 px-2 py-0.5 rounded-full">{t('apiKeys.active')}</span>
                    )}
                  </td>
                  <td className="py-2 pr-3">
                    {!k.revoked_at && (
                      <button
                        onClick={() => handleRevoke(k.id)}
                        disabled={busyId === k.id}
                        className="px-2 py-1 rounded bg-fill text-ink-3 hover:text-danger disabled:opacity-50 transition-colors"
                      >
                        {t('apiKeys.revoke')}
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
