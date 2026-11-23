import clsx from 'clsx'
import { Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { AssistantMessage } from './types'

// The request's messages. Everything is rendered as text (never HTML):
// pasted emails and model output may contain markup.
export function Conversation({ messages }: { messages: AssistantMessage[] }) {
  return (
    <ol className="flex flex-col gap-3">
      {messages.map(m => <MessageItem key={m.seq} message={m} />)}
    </ol>
  )
}

function MessageItem({ message: m }: { message: AssistantMessage }) {
  const { t } = useTranslation('assistant')

  if (m.role === 'tool') {
    return (
      <li className="flex items-center gap-1.5 text-2xs text-ink-4 pl-1">
        <Search size={11} aria-hidden /> {t('conversation.lookedUp', { tool: m.tool_name ?? '' })}
      </li>
    )
  }
  const text = m.notice ? t(`notice.${m.notice}`) : m.content
  // An assistant step that only called tools has no text of its own; its
  // tool results are listed right after it.
  if (!text) return null

  const mine = m.role === 'user'
  return (
    <li
      data-testid={`assistant-message-${m.seq}`} data-question={m.question ? 'true' : 'false'}
      className={clsx('flex flex-col gap-1 max-w-[85%]', mine ? 'self-end items-end' : 'self-start items-start')}
    >
      <span className="text-2xs text-ink-4">
        {mine ? t('conversation.you') : m.question ? t('conversation.question') : t('conversation.assistant')}
      </span>
      <p
        className={clsx(
          'text-xs whitespace-pre-wrap break-words rounded-xl px-3 py-2',
          mine && 'bg-accent-solid text-white',
          !mine && !m.question && 'bg-sunken text-ink border border-line',
          !mine && m.question && 'bg-amber-50 dark:bg-amber-500/10 text-ink border border-amber-500/40',
        )}
      >
        {text}
      </p>
    </li>
  )
}
