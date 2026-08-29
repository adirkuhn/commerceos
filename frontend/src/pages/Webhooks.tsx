import { useState } from 'react'
import { Webhook, ShieldAlert } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { useApp } from '../context/AppContext'
import { useWebhookForwardRules } from '../integrations/hooks/useWebhookForwardRules'
import { useAllProviderWebhooks } from '../integrations/hooks/useAllProviderWebhooks'
import { useWebhookTopics } from '../integrations/hooks/useWebhookTopics'
import { useWebhookEvents } from '../integrations/hooks/useWebhookEvents'
import * as api from '../integrations/api'
import type { AvailableWebhookTopics } from '../integrations/api'
import type { WebhookEvent } from '../integrations/types'

// Known providers get a friendlier display name; any future provider this
// codebase doesn't know the name of yet falls back to capitalizing what the
// backend sent -- same "provider is free text" convention as FanOutTab's
// own create form.
const KNOWN_PROVIDER_LABELS: Record<string, string> = { myparcel: 'MyParcel', woocommerce: 'WooCommerce', bol: 'Bol.com' }
function providerLabel(provider: string): string {
  return KNOWN_PROVIDER_LABELS[provider] ?? (provider.charAt(0).toUpperCase() + provider.slice(1))
}

// Same tier as WooCommerceSyncSettingsPage's webhook-secret tab and
// Integrations.tsx's undelivered-webhooks table -- gated client-side only
// (this codebase has no server-side RBAC on these endpoints either, a
// deliberate existing exception, see Integrations.tsx's own
// canManageWebhooks doc comment).
function canManageWebhooks(role: string | undefined): boolean {
  return role === 'customer_owner' || role === 'customer_admin' || role === 'super_admin'
}

type Tab = 'fanout' | 'subscriptions' | 'messages'

// The dedicated home for everything webhook-related: Screen 1 (Fan-out)
// configures CommerceOS's own fan-out rules (provider/topic -> target
// URL), Screen 2 (Subscriptions) shows every webhook subscription actually
// registered at WooCommerce/MyParcel across every connection this tenant
// has, in one place -- previously only visible one connection at a time,
// buried inside that connection's own settings page. Screen 3 (Messages)
// is the raw "see all messages" log -- every webhook actually received,
// filterable by sender.
export default function Webhooks() {
  const { token, user } = useAuth()
  const [tab, setTab] = useState<Tab>('fanout')
  const [messageProvider, setMessageProvider] = useState('')
  const allowed = canManageWebhooks(user?.role)

  const { setActiveNav } = useApp()
  const forwardRules = useWebhookForwardRules(allowed ? token ?? '' : '')
  const allWebhooks = useAllProviderWebhooks(allowed && tab === 'subscriptions' ? token ?? '' : '')
  const { topics } = useWebhookTopics(allowed ? token ?? '' : '')
  const webhookEvents = useWebhookEvents(allowed && tab === 'messages' ? token ?? '' : '', messageProvider)

  if (!allowed) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center text-center p-6">
        <ShieldAlert size={24} className="dark:text-gray-500 text-gray-400 mb-2" />
        <p className="text-sm dark:text-gray-300 text-gray-700">You don't have permission to manage webhooks.</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col p-5 gap-4">
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
        <button
          onClick={() => setTab('messages')}
          className={`px-3 py-2 text-xs font-medium border-b-2 transition-colors ${
            tab === 'messages' ? 'border-blue-500 text-blue-400' : 'border-transparent dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600'
          }`}
        >
          Messages
        </button>
      </div>

      <div>
        {tab === 'fanout' && <FanOutTab {...forwardRules} topics={topics} />}
        {tab === 'subscriptions' && <SubscriptionsTab {...allWebhooks} onRegisterWebhook={() => setActiveNav('integrations')} />}
        {tab === 'messages' && (
          <MessagesTab {...webhookEvents} topics={topics} provider={messageProvider} onProviderChange={setMessageProvider} />
        )}
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
  rules, deliveries, loading, error, create, remove, topics,
}: ReturnType<typeof useWebhookForwardRules> & { topics: AvailableWebhookTopics }) {
  const [provider, setProvider] = useState('')
  const [topic, setTopic] = useState('')
  const [targetUrl, setTargetUrl] = useState('')
  const [formError, setFormError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)

  // One optgroup per provider per section -- who's responsible for each
  // topic matters (a customer_admin picking "order.updated" needs to know
  // whether that's WooCommerce's or some future service's event), so
  // topics are never flattened across providers. Two sections regardless
  // of which providers show up: "active" is what's actually subscribed
  // right now for that provider (guaranteed to really arrive), "all
  // events" is the full reference catalog (may include topics nothing is
  // subscribed to yet). Driven entirely by the topics object's own keys,
  // so a brand-new provider gets its own groups automatically.
  function providerOptgroups(byProvider: Record<string, string[]>, suffix: string) {
    return Object.keys(byProvider).sort().map(provider => {
      const opts = Array.from(new Set(byProvider[provider])).sort()
      if (opts.length === 0) return null
      return (
        <optgroup key={`${suffix}-${provider}`} label={`${providerLabel(provider)} — ${suffix}`}>
          {opts.map(t => <option key={t} value={t}>{t}</option>)}
        </optgroup>
      )
    })
  }

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
            <input
              id="forward-provider" required value={provider}
              onChange={e => setProvider(e.target.value)}
              placeholder="woocommerce" className={inputClass}
            />
          </div>
          <div>
            <label htmlFor="forward-topic" className="block text-[11px] dark:text-gray-400 text-gray-500 mb-1">Topic</label>
            <select id="forward-topic" value={topic} onChange={e => setTopic(e.target.value)} className={inputClass}>
              <option value="">Any topic (match everything)</option>
              {providerOptgroups(topics.active_webhooks, 'active')}
              {providerOptgroups(topics.all_events, 'all events')}
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
function SubscriptionsTab({
  webhooks, loading, error, remove, onRegisterWebhook,
}: ReturnType<typeof useAllProviderWebhooks> & { onRegisterWebhook: () => void }) {
  const [busyKey, setBusyKey] = useState<string | null>(null)
  const [rowError, setRowError] = useState('')

  // A CommerceOS-registered webhook's delivery_url always hits our own
  // ingest route (see application/integrations.providerWebhookIngestURL) --
  // if none of the rows do, every subscription here points somewhere else
  // entirely (e.g. an ERP, a WordPress plugin) and CommerceOS itself isn't
  // receiving anything yet.
  const noneTargetCommerceOS = webhooks.length > 0 && webhooks.every(w => !w.delivery_url.includes('/webhooks/v1/'))

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
        <div className="flex items-start justify-between gap-3 mb-3">
          <div>
            <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-1">Active provider subscriptions</h2>
            <p className="text-xs dark:text-gray-500 text-gray-400">
              Every webhook subscription actually registered at WooCommerce/MyParcel, across every connection this tenant has --
              not necessarily pointed at CommerceOS itself.
            </p>
          </div>
          <button
            type="button" onClick={onRegisterWebhook}
            className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors shrink-0"
          >
            Register a webhook
          </button>
        </div>

        {noneTargetCommerceOS && (
          <p className="text-xs text-amber-500 bg-amber-50 dark:bg-amber-500/10 rounded-lg px-3 py-2 mb-3">
            None of these are registered with CommerceOS -- they're all pointed at other systems. Click "Register a webhook"
            above to open a connection's settings and subscribe CommerceOS too.
          </p>
        )}

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

// A message "needs attention" if it's Failed, or Received on a
// provider/topic that actually has a dispatch handler (myparcel
// shipment_status_change today) -- meaning it should have moved past
// Received but hasn't. A Received message on any other provider/topic is
// its normal, successful terminal state, not something stuck.
function needsAttention(e: WebhookEvent): boolean {
  if (e.status === 'failed') return true
  return e.status === 'received' && e.provider === 'myparcel' && e.topic === 'shipment_status_change'
}

// Screen 3: "see all messages" -- every webhook this tenant has actually
// received (backend/domain/webhooks.WebhookEvent), filterable by sender.
// Payload is truncated by default and expands inline on click -- these are
// order/shipment bodies, not huge, but a raw dump for every row would still
// bury the table. A view toggle narrows to just Failed/still-pending
// messages, each with its own Retry action (and a bulk one for the whole
// filtered set) -- re-running dispatch right now instead of waiting for
// the next webhook or a manual backfill run.
function MessagesTab({
  events, loading, error, updateEvent, topics, provider, onProviderChange,
}: ReturnType<typeof useWebhookEvents> & { topics: AvailableWebhookTopics; provider: string; onProviderChange: (provider: string) => void }) {
  const senders = Array.from(new Set([...Object.keys(topics.active_webhooks), ...Object.keys(topics.all_events)])).sort()
  const [view, setView] = useState<'all' | 'needsAttention'>('all')
  const [retryingId, setRetryingId] = useState<string | null>(null)
  const [bulkRetrying, setBulkRetrying] = useState(false)
  const [retryError, setRetryError] = useState('')
  const { token } = useAuth()

  const displayedEvents = view === 'needsAttention' ? events.filter(needsAttention) : events

  async function handleRetry(eventId: string) {
    if (!token) return
    setRetryError('')
    setRetryingId(eventId)
    try {
      const updated = await api.retryWebhookEventDispatch(token, eventId)
      updateEvent(updated)
    } catch (err) {
      setRetryError(err instanceof Error ? err.message : 'Failed to retry')
    } finally {
      setRetryingId(null)
    }
  }

  async function handleRetryAll() {
    if (!token) return
    setRetryError('')
    setBulkRetrying(true)
    try {
      // Sequential, not parallel -- this hits MyParcel's real API once per
      // message, and the dataset this view targets is small by design (see
      // needsAttention's own doc comment).
      for (const e of events.filter(needsAttention)) {
        try {
          const updated = await api.retryWebhookEventDispatch(token, e.id)
          updateEvent(updated)
        } catch (err) {
          setRetryError(err instanceof Error ? err.message : 'Failed to retry one or more messages')
        }
      }
    } finally {
      setBulkRetrying(false)
    }
  }

  return (
    <div className="max-w-4xl">
      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4">
        <div className="flex items-start justify-between gap-3 mb-3">
          <div>
            <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-1">Received messages</h2>
            <p className="text-xs dark:text-gray-500 text-gray-400">
              Every webhook this tenant has actually received, newest first.
            </p>
          </div>
          <div className="flex items-end gap-2">
            <div>
              <label htmlFor="message-sender" className="block text-[11px] dark:text-gray-400 text-gray-500 mb-1">Sender</label>
              <select
                id="message-sender" value={provider} onChange={e => onProviderChange(e.target.value)}
                className="px-2.5 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900"
              >
                <option value="">Any sender</option>
                {senders.map(s => <option key={s} value={s}>{providerLabel(s)}</option>)}
              </select>
            </div>
            {view === 'needsAttention' && displayedEvents.length > 0 && (
              <button
                type="button" onClick={handleRetryAll} disabled={bulkRetrying}
                className="px-2.5 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                {bulkRetrying ? 'Retrying…' : `Retry all (${displayedEvents.length})`}
              </button>
            )}
          </div>
        </div>

        <div className="flex gap-1 mb-3 border-b dark:border-[#2D3748] border-gray-200">
          <button
            type="button" onClick={() => setView('all')}
            className={`px-2.5 py-1.5 text-xs font-medium border-b-2 transition-colors ${
              view === 'all' ? 'border-blue-500 text-blue-400' : 'border-transparent dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600'
            }`}
          >
            All messages
          </button>
          <button
            type="button" onClick={() => setView('needsAttention')}
            className={`px-2.5 py-1.5 text-xs font-medium border-b-2 transition-colors ${
              view === 'needsAttention' ? 'border-blue-500 text-blue-400' : 'border-transparent dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600'
            }`}
          >
            Needs attention{events.filter(needsAttention).length > 0 && ` (${events.filter(needsAttention).length})`}
          </button>
        </div>

        {error && <p className="text-xs text-red-400 mb-2">{error}</p>}
        {retryError && <p className="text-xs text-red-400 mb-2">{retryError}</p>}

        {loading ? (
          <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>
        ) : displayedEvents.length === 0 ? (
          <p className="text-xs dark:text-gray-500 text-gray-400">
            {view === 'needsAttention' ? 'Nothing needs attention.' : 'No messages received yet.'}
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-left dark:text-gray-500 text-gray-400 border-b dark:border-[#2D3748] border-gray-200">
                  <th className="py-2 pr-3 font-medium">Sender</th>
                  <th className="py-2 pr-3 font-medium">Topic</th>
                  <th className="py-2 pr-3 font-medium">Received at</th>
                  <th className="py-2 pr-3 font-medium">Status</th>
                  <th className="py-2 pr-3 font-medium">Payload</th>
                  <th className="py-2 pr-3 font-medium"></th>
                </tr>
              </thead>
              <tbody>
                {displayedEvents.map(e => (
                  <tr key={e.id} className="border-b dark:border-[#2D3748] border-gray-100 last:border-0 align-top">
                    <td className="py-2 pr-3 dark:text-gray-300 text-gray-700">{providerLabel(e.provider)}</td>
                    <td className="py-2 pr-3 dark:text-gray-400 text-gray-500">{e.topic}</td>
                    <td className="py-2 pr-3 dark:text-gray-400 text-gray-500 whitespace-nowrap">{new Date(e.received_at).toLocaleString()}</td>
                    <td className="py-2 pr-3"><MessageStatusBadge status={e.status} dispatchError={e.dispatch_error} /></td>
                    <td className="py-2 pr-3 dark:text-gray-400 text-gray-500 font-mono max-w-[360px]">
                      <PayloadPreview payload={e.payload} />
                    </td>
                    <td className="py-2 pr-3">
                      {needsAttention(e) && (
                        <button
                          type="button" onClick={() => handleRetry(e.id)} disabled={retryingId === e.id || bulkRetrying}
                          className="px-2 py-1 rounded dark:bg-[#111827] bg-gray-100 dark:text-gray-400 text-gray-500 hover:dark:text-gray-200 hover:text-gray-700 disabled:opacity-50 transition-colors"
                        >
                          {retryingId === e.id ? 'Retrying…' : 'Retry'}
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
    </div>
  )
}

// "received" is the normal, successful terminal state for most events --
// no per-topic dispatch handler exists for their provider/topic, so
// nothing more was ever going to happen. Only "processed"/"failed" mean a
// dispatch handler (myparcel's shipment_status_change today) actually ran.
function MessageStatusBadge({ status, dispatchError }: { status: WebhookEvent['status']; dispatchError: string }) {
  if (status === 'failed') {
    return (
      <span
        title={dispatchError || undefined}
        className="px-1.5 py-0.5 rounded text-[11px] font-medium bg-red-500/15 text-red-500 cursor-help"
      >
        Failed
      </span>
    )
  }
  if (status === 'processed') {
    return <span className="px-1.5 py-0.5 rounded text-[11px] font-medium bg-emerald-500/15 text-emerald-500">Processed</span>
  }
  return <span className="px-1.5 py-0.5 rounded text-[11px] font-medium dark:bg-[#111827] bg-gray-100 dark:text-gray-400 text-gray-500">Received</span>
}

// Truncated one-line preview that expands into the full formatted JSON on
// click -- avoids a wall of raw JSON per row while still letting an admin
// actually see what was sent.
function PayloadPreview({ payload }: { payload: unknown }) {
  const [expanded, setExpanded] = useState(false)
  const json = JSON.stringify(payload)
  const pretty = JSON.stringify(payload, null, 2)

  if (!json || json === 'null') {
    return <span className="dark:text-gray-600 text-gray-400">—</span>
  }

  if (expanded) {
    return (
      <div>
        <pre className="whitespace-pre-wrap break-all text-[11px] dark:bg-[#111827] bg-gray-50 rounded-lg p-2 max-h-64 overflow-y-auto">{pretty}</pre>
        <button type="button" onClick={() => setExpanded(false)} className="text-blue-400 hover:underline mt-1">Collapse</button>
      </div>
    )
  }

  const preview = json.length > 60 ? `${json.slice(0, 60)}…` : json
  return (
    <button type="button" onClick={() => setExpanded(true)} className="truncate max-w-[360px] block text-left hover:underline" title="View full payload">
      {preview}
    </button>
  )
}
