import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { useProviderWebhooks } from './hooks/useProviderWebhooks'
import { apiErrorText } from '../api/apiErrorText'

// Lists what's actually registered at the provider (WooCommerce/MyParcel)
// for this connection, and lets an admin register a new subscription
// pointed at CommerceOS's own ingest URL instead of doing it by hand in
// wp-admin/the MyParcel dashboard -- shared by WooCommerceSyncSettingsPage
// and MyParcelSyncSettingsPage's own Webhooks tabs. Requires a webhook
// secret to already be generated on this connection (see the "Webhook
// signature secret" section above this one) -- Create surfaces
// ErrWebhookSecretNotSet's message inline if it isn't.
export default function ProviderWebhooksPanel({ token, connectionId }: { token: string; connectionId: string }) {
  const { webhooks, loading, error, create, remove } = useProviderWebhooks(token, connectionId)
  const { t } = useTranslation(['integrations', 'common'])
  const [topic, setTopic] = useState('')
  const [formError, setFormError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    setSubmitting(true)
    try {
      await create(topic.trim())
      setTopic('')
    } catch (err) {
      setFormError(apiErrorText(err) ?? t('providerWebhooks.createFailed'))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleRemove(id: string) {
    setBusyId(id)
    try {
      await remove(id)
    } catch (err) {
      setFormError(apiErrorText(err) ?? t('providerWebhooks.removeFailed'))
    } finally {
      setBusyId(null)
    }
  }

  return (
    <div>
      <h2 className="text-sm font-semibold text-ink mb-1">{t('providerWebhooks.title')}</h2>
      <p className="text-xs text-ink-4 mb-3">
        {t('providerWebhooks.intro')}
      </p>

      <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
        {error && <p className="text-xs text-danger">{error}</p>}
        {formError && <p className="text-xs text-danger">{formError}</p>}

        <form onSubmit={handleCreate} className="flex items-end gap-2">
          <div className="flex-1">
            <label htmlFor="provider-webhook-topic" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              {t('providerWebhooks.topic')}
            </label>
            <input
              id="provider-webhook-topic" required value={topic} onChange={e => setTopic(e.target.value)}
              placeholder={t('providerWebhooks.topicPlaceholder')}
              className="w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink"
            />
          </div>
          <button
            type="submit" disabled={submitting}
            className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {submitting ? t('creating') : t('common:actions.create')}
          </button>
        </form>

        {loading ? (
          <p className="text-xs text-ink-4">{t('common:status.loading')}</p>
        ) : webhooks.length === 0 ? (
          <p className="text-xs text-ink-4">{t('providerWebhooks.empty')}</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-left text-ink-4 border-b border-line">
                  <th className="py-2 pr-3 font-medium">{t('providerWebhooks.topic')}</th>
                  <th className="py-2 pr-3 font-medium">{t('deliveryUrl')}</th>
                  <th className="py-2 pr-3 font-medium">{t('common:fields.status')}</th>
                  <th className="py-2 pr-3 font-medium"></th>
                </tr>
              </thead>
              <tbody>
                {webhooks.map(w => (
                  <tr key={w.id} className="border-b border-line last:border-0">
                    <td className="py-2 pr-3 text-ink-2">{w.topic}</td>
                    <td className="py-2 pr-3 text-ink-3 font-mono truncate max-w-[220px]" title={w.delivery_url}>{w.delivery_url}</td>
                    <td className="py-2 pr-3 text-ink-3">{w.status}</td>
                    <td className="py-2 pr-3">
                      <button
                        onClick={() => handleRemove(w.id)}
                        disabled={busyId === w.id}
                        className="px-2 py-1 rounded bg-fill text-ink-3 hover:text-danger disabled:opacity-50 transition-colors"
                      >
                        {t('common:actions.remove')}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
