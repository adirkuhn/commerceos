import { Bot } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import { useAssistantActions } from '../hooks/useAssistantActions'
import type { AssistantRequest } from '../types'
import { Composer } from './Composer'

// An empty chat: the first message creates the request, and its own page
// opens at once (the answer arrives there).
export function NewChat({ onCreated }: { onCreated: (request: AssistantRequest) => void }) {
  const { t } = useTranslation('assistant')
  const { token } = useAuth()
  const actions = useAssistantActions(token ?? '')

  async function send(text: string): Promise<boolean> {
    const created = await actions.create(text)
    if (!created) return false
    onCreated(created)
    return true
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <div className="flex-1 min-h-0 overflow-auto flex flex-col items-center justify-center gap-2 px-5 text-center">
        <Bot size={28} className="text-accent" aria-hidden />
        <p className="text-sm font-semibold text-ink">{t('chat.emptyTitle')}</p>
        <p className="text-xs text-ink-4 max-w-md">{t('intro')}</p>
      </div>
      <div className="shrink-0 border-t border-line px-5 py-3">
        <div className="max-w-3xl mx-auto flex flex-col gap-1.5">
          {actions.error && <p className="text-xs text-danger">{actions.error}</p>}
          <Composer mode="new" busy={actions.busy} onSend={send} />
        </div>
      </div>
    </div>
  )
}
