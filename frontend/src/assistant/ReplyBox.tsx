import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'

export type ReplyMode = 'answer' | 'followUp' | 'changes'

const LABELS = { answer: 'conversation.replyLabel', followUp: 'conversation.followUpLabel', changes: 'conversation.changesLabel' } as const satisfies Record<ReplyMode, string>

// Answer the assistant's question, ask a follow-up on its answer, or ask
// for changes to what it proposed.
export function ReplyBox({ mode, busy, onSend }: { mode: ReplyMode; busy: boolean; onSend: (text: string) => Promise<boolean> }) {
  const { t } = useTranslation('assistant')
  const [text, setText] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!text.trim()) return
    if (await onSend(text)) setText('')
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-2">
      <label htmlFor="assistant-reply" className="text-2xs text-ink-4 caps-label">
        {t(LABELS[mode])}
      </label>
      <textarea
        id="assistant-reply" rows={3} value={text} onChange={e => setText(e.target.value)} maxLength={20000}
        placeholder={t('conversation.replyPlaceholder')} autoFocus={mode === 'answer'}
        className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink outline-none focus:ring-1 focus:ring-focus resize-y"
      />
      <div>
        <button
          type="submit" disabled={busy || !text.trim()}
          className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50"
        >
          {busy ? t('conversation.sending') : t('conversation.send')}
        </button>
      </div>
    </form>
  )
}
