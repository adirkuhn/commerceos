import { useState } from 'react'
import { Webhook, ShieldAlert } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { useWebhookForwardRules } from '../integrations/hooks/useWebhookForwardRules'
import { useAllProviderWebhooks } from '../integrations/hooks/useAllProviderWebhooks'

// Same tier as WooCommerceSyncSettingsPage's webhook-secret tab and
// Integrations.tsx's undelivered-webhooks table -- gated client-side only
// (this codebase has no server-side RBAC on these endpoints either, a
// deliberate existing exception, see Integrations.tsx's own
// canManageWebhooks doc comment).
function canManageWebhooks(role: string | undefined): boolean {
  return role === 'customer_owner' || role === 'customer_admin' || role === 'super_admin'
}

type Tab = 'fanout' | 'subscriptions'

// Fixed topic lists per provider, so the Topic field can be a select
// instead of free text -- WooCommerce's own REST API webhook topics
// (Woo docs: Webhooks -> Topics) and MyParcel's three hook types (see
// backend/domain/integrations.WebhookSubscription's doc comment,
// confirmed against developer.myparcel.nl/api-reference/10.webhooks.html).
// Not validated against these on the backend -- Woo/MyParcel themselves
// reject an unknown one -- this list only drives the dropdown.
const TOPICS_BY_PROVIDER: Record<string, string[]> = {
  woocommerce: [
    'order.created', 'order.updated', 'order.deleted',
    'product.created', 'product.updated', 'product.deleted',
    'customer.created', 'customer.updated', 'customer.deleted',
    'coupon.created', 'coupon.updated', 'coupon.deleted',
  ],
  myparcel: ['shipment_status_change', 'shipment_label_created', 'order_status_change'],
}

// The dedicated home for everything webhook-related: Screen 1 (Fan-out)
// configures CommerceOS's own fan-out rules (provider/topic -> target
// URL), Screen 2 (Subscriptions) shows every webhook subscription actually
// registered at WooCommerce/MyParcel across every connection this tenant
// has, in one place -- previously only visible one connection at a time,
// buried inside that connection's own settings page.
export default function Webhooks() {
  const { token, user } = useAuth()
  const [tab, setTab] = useState<Tab>('fanout')
  const allowed = canManageWebhooks(user?.role)

  const forwardRules = useWebhookForwardRules(allowed ? token ?? '' : '')
  const allWebhooks = useAllProviderWebhooks(allowed && tab === 'subscriptions' ? token ?? '' : '')

  if (!allowed) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center text-center p-6">
        <ShieldAlert size={24} className="dark:text-gray-500 text-gray-400 mb-2" />
        <p className="text-sm dark:text-gray-300 text-gray-700">You don't have permission to manage webhooks.</p>
      </div>
    )
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden p-5 gap-4">
      <div className="flex items-center gap-2.5">
        <div className="w-8 h-8 rounded-lg dark:bg-blue-600/15 bg-blue-50 flex items-center justify-center shrink-0">
          <Webhook size={15} className="text-blue-400" />
        </div>
        <h1 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Webhooks</h1>
      </div>

      <div className="flex gap-1 border-b dark:border-[#2D3748] border-gray-200">
        <button
          onClick={() => setTab('fanout')}
          className={`px-3 py-2 text-xs font-medium border-b-2 transition-colors ${
            tab === 'fanout' ? 'border-blue-500 text-blue-400' : 'border-transparent dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600'
          }`}
        >
          Fan-out
        </button>
        <button
          onClick={() => setTab('subscriptions')}
          className={`px-3 py-2 text-xs font-medium border-b-2 transition-colors ${
            tab === 'subscriptions' ? 'border-blue-500 text-blue-400' : 'border-transparent dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600'
          }`}
        >
          Subscriptions
        </button>
      </div>

      <div className="flex-1 overflow-y-auto">
        {tab === 'fanout' && <FanOutTab {...forwardRules} />}
        {tab === 'subscriptions' && <SubscriptionsTab {...allWebhooks} />}
      </div>
    </div>
  )
}

// Screen 1: CommerceOS's own fan-out configuration -- "also POST every
// provider/topic webhook to this URL". Topic left blank matches every
// topic for that provider (see backend/domain/webhooks.ForwardRule's doc
// comment). The create form always shows (unlike the tables below it,
// which hide themselves when empty) since this is a config surface, not
// just an alert list.
function FanOutTab({
  rules, deliveries, loading, error, create, remove,
}: ReturnType<typeof useWebhookForwardRules>) {
  const [provider, setProvider] = useState('')
  const [topic, setTopic] = useState('')
  const [targetUrl, setTargetUrl] = useState('')
  const [formError, setFormError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    setSubmitting(true)
    try {
      await create({ provider: provider.trim(), topic: topic.trim() || undefined, target_url: targetUrl.trim() })
      setProvider('')
      setTopic('')
      setTargetUrl('')
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to add rule')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleRemove(id: string) {
    setBusyId(id)
    try {
      await remove(id)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to remove rule')
    } finally {
      setBusyId(null)
    }
  }

  const inputClass = 'px-2.5 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900'

  return (
    <div className="max-w-4xl space-y-4">
      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4">
        <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-1">Webhook forwarding</h2>
        <p className="text-xs dark:text-gray-500 text-gray-400 mb-3">
          Fan out a received webhook to other systems -- every event matching provider/topic is also POSTed to the target
          URL (its exact original payload, byte for byte). Leave topic blank to match every event for that provider.
        </p>

        {error && <p className="text-xs text-red-400 mb-2">{error}</p>}
        {formError && <p className="text-xs text-red-400 mb-2">{formError}</p>}

        <form onSubmit={handleCreate} className="flex flex-wrap items-end gap-2 mb-4">
          <div>
            <label htmlFor="forward-provider" className="block text-[11px] dark:text-gray-400 text-gray-500 mb-1">Provider</label>
            <select
              id="forward-provider" required value={provider}
              onChange={e => { setProvider(e.target.value); setTopic('') }}
              className={inputClass}
            >
              <option value="">Select…</option>
              <option value="woocommerce">WooCommerce</option>
              <option value="myparcel">MyParcel</option>
            </select>
          </div>
          <div>
            <label htmlFor="forward-topic" className="block text-[11px] dark:text-gray-400 text-gray-500 mb-1">Topic</label>
            <select id="forward-topic" value={topic} onChange={e => setTopic(e.target.value)} className={inputClass}>
              <option value="">All events</option>
              {(TOPICS_BY_PROVIDER[provider] ?? []).map(t => (
                <option key={t} value={t}>{t}</option>
              ))}
            </select>
          </div>
          <div className="flex-1 min-w-[220px]">
            <label htmlFor="forward-target-url" className="block text-[11px] dark:text-gray-400 text-gray-500 mb-1">Target URL</label>
            <input id="forward-target-url" required value={targetUrl} onChange={e => setTargetUrl(e.target.value)} placeholder="https://shop.example.com/hook" className={`${inputClass} w-full`} />
          </div>
          <button
            type="submit" disabled={submitting}
            className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {submitting ? 'Adding…' : 'Add rule'}
          </button>
        </form>

        {loading ? (
          <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>
        ) : rules.length === 0 ? (
          <p className="text-xs dark:text-gray-500 text-gray-400">No forwarding rules configured yet.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-left dark:text-gray-500 text-gray-400 border-b dark:border-[#2D3748] border-gray-200">
                  <th className="py-2 pr-3 font-medium">Provider</th>
                  <th className="py-2 pr-3 font-medium">Topic</th>
                  <th className="py-2 pr-3 font-medium">Target URL</th>
                  <th className="py-2 pr-3 font-medium"></th>
                </tr>
              </thead>
              <tbody>
                {rules.map(r => (
                  <tr key={r.id} className="border-b dark:border-[#2D3748] border-gray-100 last:border-0">
                    <td className="py-2 pr-3 dark:text-gray-300 text-gray-700 capitalize">{r.provider}</td>
                    <td className="py-2 pr-3 dark:text-gray-400 text-gray-500">{r.topic || 'any'}</td>
                    <td className="py-2 pr-3 dark:text-gray-400 text-gray-500 font-mono truncate max-w-[280px]" title={r.target_url}>{r.target_url}</td>
                    <td className="py-2 pr-3">
                      <button
                        onClick={() => handleRemove(r.id)}
                        disabled={busyId === r.id}
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

      {deliveries.length > 0 && (
        <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4">
          <h3 className="text-xs font-semibold dark:text-gray-300 text-gray-700 mb-2">Recent deliveries</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-left dark:text-gray-500 text-gray-400 border-b dark:border-[#2D3748] border-gray-200">
                  <th className="py-2 pr-3 font-medium">Provider / Topic</th>
                  <th className="py-2 pr-3 font-medium">Target URL</th>
                  <th className="py-2 pr-3 font-medium">Status</th>
                  <th className="py-2 pr-3 font-medium">Attempts</th>
                </tr>
              </thead>
              <tbody>
                {deliveries.map(d => (
                  <tr key={d.id} className="border-b dark:border-[#2D3748] border-gray-100 last:border-0">
                    <td className="py-2 pr-3 dark:text-gray-300 text-gray-700">{d.provider} / {d.topic || 'any'}</td>
                    <td className="py-2 pr-3 dark:text-gray-400 text-gray-500 font-mono truncate max-w-[240px]" title={d.target_url}>{d.target_url}</td>
                    <td className="py-2 pr-3 dark:text-gray-400 text-gray-500" title={d.last_error}>{d.status}</td>
                    <td className="py-2 pr-3 dark:text-gray-400 text-gray-500">{d.attempts}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}

// Screen 2: what's actually subscribed right now, across every connection
// -- the "give me visibility" ask. Read-only list + Remove; Create still
// lives on each connection's own settings page (WooCommerceSyncSettingsPage
// / MyParcelSyncSettingsPage's Webhooks tab), since creating one requires
// picking which connection it belongs to and that page already has that
// context, plus the webhook-secret generation step right next to it.
function SubscriptionsTab({ webhooks, loading, error, remove }: ReturnType<typeof useAllProviderWebhooks>) {
  const [busyKey, setBusyKey] = useState<string | null>(null)
  const [rowError, setRowError] = useState('')

  async function handleRemove(connectionId: string, webhookId: string) {
    const key = `${connectionId}:${webhookId}`
    setRowError('')
    setBusyKey(key)
    try {
      await remove(connectionId, webhookId)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to remove subscription')
    } finally {
      setBusyKey(null)
    }
  }

  return (
    <div className="max-w-4xl">
      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4">
        <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-1">Active provider subscriptions</h2>
        <p className="text-xs dark:text-gray-500 text-gray-400 mb-3">
          Every webhook subscription actually registered at WooCommerce/MyParcel, across every connection this tenant has.
          To add a new one, open that connection's own settings page (Webhooks tab).
        </p>

        {error && <p className="text-xs text-red-400 mb-2">{error}</p>}
        {rowError && <p className="text-xs text-red-400 mb-2">{rowError}</p>}

        {loading ? (
          <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>
        ) : webhooks.length === 0 ? (
          <p className="text-xs dark:text-gray-500 text-gray-400">Nothing registered yet at any connection.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-left dark:text-gray-500 text-gray-400 border-b dark:border-[#2D3748] border-gray-200">
                  <th className="py-2 pr-3 font-medium">Connection</th>
                  <th className="py-2 pr-3 font-medium">Topic</th>
                  <th className="py-2 pr-3 font-medium">Delivery URL</th>
                  <th className="py-2 pr-3 font-medium">Status</th>
                  <th className="py-2 pr-3 font-medium"></th>
                </tr>
              </thead>
              <tbody>
                {webhooks.map(w => {
                  const key = `${w.connection_id}:${w.id}`
                  return (
                    <tr key={key} className="border-b dark:border-[#2D3748] border-gray-100 last:border-0">
                      <td className="py-2 pr-3 dark:text-gray-300 text-gray-700">
                        <span className="capitalize">{w.provider}</span>
                        {w.store_label && <span className="dark:text-gray-500 text-gray-400"> · {w.store_label}</span>}
                      </td>
                      <td className="py-2 pr-3 dark:text-gray-400 text-gray-500">{w.topic || 'any'}</td>
                      <td className="py-2 pr-3 dark:text-gray-400 text-gray-500 font-mono truncate max-w-[240px]" title={w.delivery_url}>{w.delivery_url}</td>
                      <td className="py-2 pr-3 dark:text-gray-400 text-gray-500">{w.status}</td>
                      <td className="py-2 pr-3">
                        <button
                          onClick={() => w.connection_id && handleRemove(w.connection_id, w.id)}
                          disabled={busyKey === key || !w.connection_id}
                          className="px-2 py-1 rounded dark:bg-[#111827] bg-gray-100 dark:text-gray-400 text-gray-500 hover:text-red-400 disabled:opacity-50 transition-colors"
                        >
                          Remove
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
