import { useTranslation } from 'react-i18next'

import { ShieldAlert } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { useApp } from '../context/AppContext'
import { useWebhookForwardRules } from '../integrations/hooks/useWebhookForwardRules'
import { useAllProviderWebhooks } from '../integrations/hooks/useAllProviderWebhooks'
import { useWebhookTopics } from '../integrations/hooks/useWebhookTopics'
import { useWebhookEvents } from '../integrations/hooks/useWebhookEvents'
import { useLocation, useNavigate } from 'react-router-dom'
import { subPathSegments } from '../routes'
import { useQueryString } from '../routing/useQueryState'
import { FanOutTab } from '../webhooks/FanOutTab'
import { SubscriptionsTab } from '../webhooks/SubscriptionsTab'
import { MessagesTab } from '../webhooks/MessagesTab'

// Same tier as WooCommerceSyncSettingsPage's webhook-secret tab and
// Integrations.tsx's undelivered-webhooks table -- gated client-side only
// (this codebase has no server-side RBAC on these endpoints either, a
// deliberate existing exception, see Integrations.tsx's own
// canManageWebhooks doc comment).
function canManageWebhooks(role: string | undefined): boolean {
  return role === 'customer_owner' || role === 'customer_admin' || role === 'super_admin'
}

const WEBHOOKS_PATH = '/webhooks'
const TABS = ['fanout', 'subscriptions', 'messages'] as const
type Tab = typeof TABS[number]
const DEFAULT_TAB: Tab = 'fanout'

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
  const { t } = useTranslation(['webhooks', 'common'])
  // Tab is the path (/webhooks/<tab>); the Messages tab's sender filter
  // and view are query params -- refresh and Back keep them.
  const location = useLocation()
  const navigate = useNavigate()
  const [segment] = subPathSegments(location.pathname, WEBHOOKS_PATH)
  const tab: Tab = (TABS as readonly string[]).includes(segment) ? segment as Tab : DEFAULT_TAB
  const setTab = (next: Tab) => {
    if (next !== tab) navigate(next === DEFAULT_TAB ? WEBHOOKS_PATH : `${WEBHOOKS_PATH}/${next}`)
  }
  const [senderParam, setMessageProvider] = useQueryString('sender')
  const messageProvider = senderParam ?? ''
  const allowed = canManageWebhooks(user?.role)

  const { setActiveNav } = useApp()
  const forwardRules = useWebhookForwardRules(allowed ? token ?? '' : '')
  const allWebhooks = useAllProviderWebhooks(allowed && tab === 'subscriptions' ? token ?? '' : '')
  const { topics } = useWebhookTopics(allowed ? token ?? '' : '')
  const webhookEvents = useWebhookEvents(allowed && tab === 'messages' ? token ?? '' : '', messageProvider)

  if (!allowed) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center text-center p-6">
        <ShieldAlert size={24} className="text-ink-4 mb-2" />
        <p className="text-sm text-ink-2">{t('noPermission')}</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col p-5 gap-4">
      <div className="flex gap-1 border-b border-line">
        <button aria-pressed={tab === 'fanout'}
          onClick={() => setTab('fanout')}
          className={`px-3 py-2 text-xs font-medium border-b-2 transition-colors ${
            tab === 'fanout' ? 'border-accent text-accent' : 'border-transparent text-ink-4 hover:text-ink-2'
          }`}
        >
          {t('tabs.fanout')}
        </button>
        <button aria-pressed={tab === 'subscriptions'}
          onClick={() => setTab('subscriptions')}
          className={`px-3 py-2 text-xs font-medium border-b-2 transition-colors ${
            tab === 'subscriptions' ? 'border-accent text-accent' : 'border-transparent text-ink-4 hover:text-ink-2'
          }`}
        >
          {t('tabs.subscriptions')}
        </button>
        <button aria-pressed={tab === 'messages'}
          onClick={() => setTab('messages')}
          className={`px-3 py-2 text-xs font-medium border-b-2 transition-colors ${
            tab === 'messages' ? 'border-accent text-accent' : 'border-transparent text-ink-4 hover:text-ink-2'
          }`}
        >
          {t('tabs.messages')}
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
