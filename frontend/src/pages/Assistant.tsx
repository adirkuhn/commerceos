import { Bot, MessagesSquare, ShieldAlert } from 'lucide-react'
import { useCallback, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { idFromPath } from '../routes'
import { ChatHistory } from '../assistant/chat/ChatHistory'
import { ChatThread } from '../assistant/chat/ChatThread'
import { NewChat } from '../assistant/chat/NewChat'
import { ASSISTANT_PATH, assistantRequestPath, canUseAssistant } from '../assistant/format'
import { useAssistantActions } from '../assistant/hooks/useAssistantActions'
import { useAssistantRequests } from '../assistant/hooks/useAssistantRequests'
import type { AssistantRequest } from '../assistant/types'
import { useCancelRequest } from '../assistant/useCancelRequest'

// The admin AI assistant as a chat: the history beside the open chat.
// /assistant is a new chat, /assistant/<id> that chat. Admin roles only,
// like the API.
export default function Assistant() {
  const { t } = useTranslation('assistant')
  const { user } = useAuth()

  if (!canUseAssistant(user?.role)) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-2 text-ink-4">
        <ShieldAlert size={24} />
        <p className="text-sm">{t('adminOnly')}</p>
      </div>
    )
  }
  return <AssistantChat />
}

function AssistantChat() {
  const { t } = useTranslation('assistant')
  const { token } = useAuth()
  const { pathname } = useLocation()
  const navigate = useNavigate()
  const id = idFromPath(pathname, ASSISTANT_PATH)
  const list = useAssistantRequests(token ?? '', undefined)
  const actions = useAssistantActions(token ?? '')
  const [showHistory, setShowHistory] = useState(false)
  const [refreshSignal, setRefreshSignal] = useState(0)

  const changedFromHistory = useCallback((r: AssistantRequest) => {
    list.replace(r)
    if (r.id === id) setRefreshSignal(n => n + 1)
  }, [list, id])
  const { cancel, confirmDialog } = useCancelRequest(actions, changedFromHistory)

  async function retry(requestId: string) {
    const out = await actions.retry(requestId)
    if (out) changedFromHistory(out)
  }

  function created(r: AssistantRequest) {
    list.upsert(r)
    navigate(assistantRequestPath(r.id))
  }

  return (
    <div className="flex flex-col h-full min-h-0">
      <div className="flex items-center gap-2 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <Bot size={14} className="text-accent" />
        <h1 className="text-sm font-semibold text-ink flex-1">{t('title')}</h1>
        <button
          type="button" onClick={() => setShowHistory(s => !s)} aria-expanded={showHistory}
          className="md:hidden flex items-center gap-1.5 px-2 py-1 rounded-lg border border-line text-xs text-ink-2"
        >
          <MessagesSquare size={13} aria-hidden /> {t('chat.history')}
        </button>
      </div>
      {actions.error && <p className="px-5 py-1.5 text-xs text-danger border-b border-line">{actions.error}</p>}
      <div className="flex flex-1 min-h-0">
        <ChatHistory
          list={list} activeId={id} className={showHistory ? 'flex' : 'hidden md:flex'}
          onNewChat={() => { setShowHistory(false); navigate(ASSISTANT_PATH) }}
          onOpen={() => setShowHistory(false)} onCancel={cancel} onRetry={retry}
        />
        <main className="flex-1 min-w-0 flex flex-col min-h-0">
          {id
            ? <ChatThread key={id} id={id} onRequest={list.upsert} refreshSignal={refreshSignal} />
            : <NewChat onCreated={created} />}
        </main>
      </div>
      {confirmDialog}
    </div>
  )
}
