import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { useWebhookForwardRules } from '../integrations/hooks/useWebhookForwardRules'
import type { AvailableWebhookTopics } from '../integrations/api'
import { providerLabel } from './providerLabel'

// Screen 1: CommerceOS's own fan-out configuration -- "also POST every
// provider/topic webhook to this URL". Topic left blank matches every
// topic for that provider (see backend/domain/webhooks.ForwardRule's doc
// comment). The create form always shows (unlike the tables below it,
// which hide themselves when empty) since this is a config surface, not
// just an alert list.
export function FanOutTab({
  rules, deliveries, loading, error, create, remove, topics,
}: ReturnType<typeof useWebhookForwardRules> & { topics: AvailableWebhookTopics }) {
  const { t } = useTranslation(['webhooks', 'common'])
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
  function providerOptgroups(byProvider: Record<string, string[]>, suffix: string, suffixLabel: string) {
    return Object.keys(byProvider).sort().map(provider => {
      const opts = Array.from(new Set(byProvider[provider])).sort()
      if (opts.length === 0) return null
      return (
        <optgroup key={`${suffix}-${provider}`} label={`${providerLabel(provider)} — ${suffixLabel}`}>
          {opts.map(o => <option key={o} value={o}>{o}</option>)}
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
      setFormError(err instanceof Error ? err.message : t('fanout.addFailed'))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleRemove(id: string) {
    setBusyId(id)
    try {
      await remove(id)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : t('fanout.removeFailed'))
    } finally {
      setBusyId(null)
    }
  }

  const inputClass = 'px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink'

  return (
    <div className="max-w-4xl space-y-4">
      <div className="bg-raised rounded-xl border border-line p-4">
        <h2 className="text-sm font-semibold text-ink mb-1">{t('fanout.title')}</h2>
        <p className="text-xs text-ink-4 mb-3">
          {t('fanout.intro')}
        </p>

        {error && <p className="text-xs text-danger mb-2">{error}</p>}
        {formError && <p className="text-xs text-danger mb-2">{formError}</p>}

        <form onSubmit={handleCreate} className="flex flex-wrap items-end gap-2 mb-4">
          <div>
            <label htmlFor="forward-provider" className="block text-2xs text-ink-3 mb-1">{t('provider')}</label>
            <input
              id="forward-provider" required value={provider}
              onChange={e => setProvider(e.target.value)}
              placeholder={'woocommerce'} className={inputClass}
            />
          </div>
          <div>
            <label htmlFor="forward-topic" className="block text-2xs text-ink-3 mb-1">{t('topic')}</label>
            <select id="forward-topic" value={topic} onChange={e => setTopic(e.target.value)} className={inputClass}>
              <option value="">{t('fanout.anyTopic')}</option>
              {providerOptgroups(topics.active_webhooks, 'active', t('fanout.groupActive'))}
              {providerOptgroups(topics.all_events, 'all events', t('fanout.groupAll'))}
            </select>
          </div>
          <div className="flex-1 min-w-[220px]">
            <label htmlFor="forward-target-url" className="block text-2xs text-ink-3 mb-1">{t('targetUrl')}</label>
            <input id="forward-target-url" required value={targetUrl} onChange={e => setTargetUrl(e.target.value)} placeholder={'https://shop.example.com/hook'} className={`${inputClass} w-full`} />
          </div>
          <button
            type="submit" disabled={submitting}
            className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {submitting ? t('fanout.adding') : t('fanout.add')}
          </button>
        </form>

        {loading ? (
          <p className="text-xs text-ink-4">{t('common:status.loading')}</p>
        ) : rules.length === 0 ? (
          <p className="text-xs text-ink-4">{t('fanout.empty')}</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-left text-ink-4 border-b border-line">
                  <th className="py-2 pr-3 font-medium">{t('provider')}</th>
                  <th className="py-2 pr-3 font-medium">{t('topic')}</th>
                  <th className="py-2 pr-3 font-medium">{t('targetUrl')}</th>
                  <th className="py-2 pr-3 font-medium"></th>
                </tr>
              </thead>
              <tbody>
                {rules.map(r => (
                  <tr key={r.id} className="border-b border-line last:border-0">
                    <td className="py-2 pr-3 text-ink-2 capitalize">{r.provider}</td>
                    <td className="py-2 pr-3 text-ink-3">{r.topic || t('any')}</td>
                    <td className="py-2 pr-3 text-ink-3 font-mono truncate max-w-[280px]" title={r.target_url}>{r.target_url}</td>
                    <td className="py-2 pr-3">
                      <button
                        onClick={() => handleRemove(r.id)}
                        disabled={busyId === r.id}
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

      {deliveries.length > 0 && (
        <div className="bg-raised rounded-xl border border-line p-4">
          <h3 className="text-xs font-semibold text-ink-2 mb-2">{t('fanout.recent')}</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-left text-ink-4 border-b border-line">
                  <th className="py-2 pr-3 font-medium">{t('fanout.providerTopic')}</th>
                  <th className="py-2 pr-3 font-medium">{t('targetUrl')}</th>
                  <th className="py-2 pr-3 font-medium">{t('common:fields.status')}</th>
                  <th className="py-2 pr-3 font-medium">{t('fanout.attempts')}</th>
                </tr>
              </thead>
              <tbody>
                {deliveries.map(d => (
                  <tr key={d.id} className="border-b border-line last:border-0">
                    <td className="py-2 pr-3 text-ink-2">{d.provider} / {d.topic || t('any')}</td>
                    <td className="py-2 pr-3 text-ink-3 font-mono truncate max-w-[240px]" title={d.target_url}>{d.target_url}</td>
                    <td className="py-2 pr-3 text-ink-3" title={d.last_error}>{d.status}</td>
                    <td className="py-2 pr-3 text-ink-3">{d.attempts}</td>
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
