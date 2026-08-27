import { useState } from 'react'
import { useProviderWebhooks } from './hooks/useProviderWebhooks'

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
      setFormError(err instanceof Error ? err.message : 'Failed to create subscription')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleRemove(id: string) {
    setBusyId(id)
    try {
      await remove(id)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to remove subscription')
    } finally {
      setBusyId(null)
    }
  }

  return (
    <div>
      <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-1">Provider webhook subscriptions</h2>
      <p className="text-xs dark:text-gray-500 text-gray-400 mb-3">
        What's actually registered at the provider, pointed at the delivery URL above. Create registers a new
        subscription for a topic instead of doing it by hand in the provider's own admin UI.
      </p>

      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
        {error && <p className="text-xs text-red-400">{error}</p>}
        {formError && <p className="text-xs text-red-400">{formError}</p>}

        <form onSubmit={handleCreate} className="flex items-end gap-2">
          <div className="flex-1">
            <label htmlFor="provider-webhook-topic" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
              Topic
            </label>
            <input
              id="provider-webhook-topic" required value={topic} onChange={e => setTopic(e.target.value)}
              placeholder="e.g. order.updated"
              className="w-full px-2.5 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900"
            />
          </div>
          <button
            type="submit" disabled={submitting}
            className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {submitting ? 'Creating…' : 'Create'}
          </button>
        </form>

        {loading ? (
          <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>
        ) : webhooks.length === 0 ? (
          <p className="text-xs dark:text-gray-500 text-gray-400">No subscriptions registered yet.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-left dark:text-gray-500 text-gray-400 border-b dark:border-[#2D3748] border-gray-200">
                  <th className="py-2 pr-3 font-medium">Topic</th>
                  <th className="py-2 pr-3 font-medium">Delivery URL</th>
                  <th className="py-2 pr-3 font-medium">Status</th>
                  <th className="py-2 pr-3 font-medium"></th>
                </tr>
              </thead>
              <tbody>
                {webhooks.map(w => (
                  <tr key={w.id} className="border-b dark:border-[#2D3748] border-gray-100 last:border-0">
                    <td className="py-2 pr-3 dark:text-gray-300 text-gray-700">{w.topic}</td>
                    <td className="py-2 pr-3 dark:text-gray-400 text-gray-500 font-mono truncate max-w-[220px]" title={w.delivery_url}>{w.delivery_url}</td>
                    <td className="py-2 pr-3 dark:text-gray-400 text-gray-500">{w.status}</td>
                    <td className="py-2 pr-3">
                      <button
                        onClick={() => handleRemove(w.id)}
                        disabled={busyId === w.id}
                        className="px-2 py-1 rounded dark:bg-[#111827] bg-gray-100 dark:text-gray-400 text-gray-500 hover:text-red-400 disabled:opacity-50 transition-colors"
                      >
                        Remove
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
