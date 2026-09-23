import { useLocation, useNavigate } from 'react-router-dom'
import clsx from 'clsx'
import { subPathSegments } from '../routes'
import {
  MESSAGE_CENTER_PATH, MESSAGE_CENTER_TABS, DEFAULT_MESSAGE_CENTER_TAB, messageCenterTabPath, type MessageCenterTab,
} from '../messagecenter/messageCenterPaths'
import MessageTemplatesPanel from '../messagecenter/MessageTemplatesPanel'
import AutomationRulesPanel from '../messagecenter/AutomationRulesPanel'
import WhatsAppConversationsPanel from '../whatsapp/WhatsAppConversationsPanel'

type Tab = MessageCenterTab

// The tab is the first URL segment (see messageCenterPaths) -- refresh
// keeps it, Back returns to the previous tab.
export default function MessageCenter() {
  const location = useLocation()
  const navigate = useNavigate()
  const [segment] = subPathSegments(location.pathname, MESSAGE_CENTER_PATH)
  const tab: Tab = (MESSAGE_CENTER_TABS as readonly string[]).includes(segment) ? segment as Tab : DEFAULT_MESSAGE_CENTER_TAB
  const setTab = (next: Tab) => {
    if (next !== tab) navigate(messageCenterTabPath(next))
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-1 px-5 pt-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <button
          onClick={() => setTab('templates')}
          className={clsx(
            'px-3 py-2 text-xs font-medium border-b-2 transition-colors',
            tab === 'templates'
              ? 'border-blue-500 dark:text-gray-100 text-gray-900'
              : 'border-transparent dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600',
          )}
        >
          Templates
        </button>
        <button
          onClick={() => setTab('rules')}
          className={clsx(
            'px-3 py-2 text-xs font-medium border-b-2 transition-colors',
            tab === 'rules'
              ? 'border-blue-500 dark:text-gray-100 text-gray-900'
              : 'border-transparent dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600',
          )}
        >
          Automation Rules
        </button>
        <button
          onClick={() => setTab('conversations')}
          className={clsx(
            'px-3 py-2 text-xs font-medium border-b-2 transition-colors',
            tab === 'conversations'
              ? 'border-blue-500 dark:text-gray-100 text-gray-900'
              : 'border-transparent dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600',
          )}
        >
          Conversations
        </button>
      </div>

      <div className="flex-1 min-h-0">
        {tab === 'templates' ? <MessageTemplatesPanel /> : tab === 'rules' ? <AutomationRulesPanel /> : <WhatsAppConversationsPanel />}
      </div>
    </div>
  )
}
