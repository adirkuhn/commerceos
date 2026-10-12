import { useLocation, useNavigate } from 'react-router-dom'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { subPathSegments } from '../routes'
import {
  MESSAGE_CENTER_PATH, MESSAGE_CENTER_TABS, DEFAULT_MESSAGE_CENTER_TAB, messageCenterTabPath, type MessageCenterTab,
} from '../messagecenter/messageCenterPaths'
import MessageTemplatesPanel from '../messagecenter/MessageTemplatesPanel'
import AutomationRulesPanel from '../messagecenter/AutomationRulesPanel'
import WhatsAppConversationsPanel from '../whatsapp/WhatsAppConversationsPanel'
import SentEmailsPanel from '../messagecenter/sent/SentEmailsPanel'

type Tab = MessageCenterTab

// The tab is the first URL segment (see messageCenterPaths) -- refresh
// keeps it, Back returns to the previous tab.
export default function MessageCenter() {
  const location = useLocation()
  const navigate = useNavigate()
  const { t } = useTranslation('messages')
  const [segment] = subPathSegments(location.pathname, MESSAGE_CENTER_PATH)
  const tab: Tab = (MESSAGE_CENTER_TABS as readonly string[]).includes(segment) ? segment as Tab : DEFAULT_MESSAGE_CENTER_TAB
  const TAB_LABELS: Record<Tab, string> = {
    templates: t('tabs.templates'), rules: t('tabs.rules'), sent: t('tabs.sent'), conversations: t('tabs.conversations'),
  }
  const setTab = (next: Tab) => {
    if (next !== tab) navigate(messageCenterTabPath(next))
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-1 px-5 pt-3 border-b border-line bg-sunken shrink-0">
        {MESSAGE_CENTER_TABS.map(id => (
          <button key={id} aria-pressed={tab === id}
            onClick={() => setTab(id)}
            className={clsx(
              'px-3 py-2 text-xs font-medium border-b-2 transition-colors',
              tab === id
                ? 'border-accent text-ink'
                : 'border-transparent text-ink-4 hover:text-ink-2',
            )}
          >
            {TAB_LABELS[id]}
          </button>
        ))}
      </div>

      <div className="flex-1 min-h-0">
        {tab === 'templates' && <MessageTemplatesPanel />}
        {tab === 'rules' && <AutomationRulesPanel />}
        {tab === 'sent' && <SentEmailsPanel />}
        {tab === 'conversations' && <WhatsAppConversationsPanel />}
      </div>
    </div>
  )
}
