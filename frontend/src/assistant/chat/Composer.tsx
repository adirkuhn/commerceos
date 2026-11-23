import { useLayoutEffect, useRef, useState, type FormEvent, type KeyboardEvent } from 'react'
import { useTranslation } from 'react-i18next'

export type ComposerMode = 'new' | 'answer' | 'followUp' | 'changes'

const LABELS = {
  new: 'compose.label', answer: 'conversation.replyLabel', followUp: 'conversation.followUpLabel', changes: 'conversation.changesLabel',
} as const satisfies Record<ComposerMode, string>

const MAX_HEIGHT_PX = 240

interface Props {
  mode: ComposerMode
  busy: boolean
  // The assistant is working: nothing can be sent until it answers.
  disabled?: boolean
  hint?: string
  onSend: (text: string) => Promise<boolean>
}

// The chat's input, pinned under the thread: Enter sends, Shift+Enter adds
// a line, and it grows with what's typed.
export function Composer({ mode, busy, disabled = false, hint, onSend }: Props) {
  const { t } = useTranslation('assistant')
  const [text, setText] = useState('')
  const box = useRef<HTMLTextAreaElement>(null)

  useLayoutEffect(() => {
    const el = box.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${Math.min(el.scrollHeight, MAX_HEIGHT_PX)}px`
  }, [text])

  async function send() {
    if (disabled || busy || !text.trim()) return
    if (await onSend(text)) setText('')
  }

  function submit(e: FormEvent) {
    e.preventDefault()
    send()
  }

  function onKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault()
      send()
    }
  }

  const isNew = mode === 'new'
  return (
    <form onSubmit={submit} className="flex flex-col gap-1.5">
      <div className="flex items-end gap-2 rounded-xl border border-line bg-surface px-3 py-2 focus-within:ring-1 focus-within:ring-focus">
        <label htmlFor="assistant-composer" className="sr-only">{t(LABELS[mode])}</label>
        <textarea
          id="assistant-composer" ref={box} rows={1} value={text} maxLength={20000} disabled={disabled}
          onChange={e => setText(e.target.value)} onKeyDown={onKeyDown}
          placeholder={isNew ? t('compose.placeholder') : t(LABELS[mode])} autoFocus={mode === 'answer' || isNew}
          className="flex-1 min-w-0 resize-none bg-transparent text-xs text-ink outline-none disabled:opacity-60 py-1"
        />
        <button
          type="submit" disabled={disabled || busy || !text.trim()}
          className="shrink-0 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50"
        >
          {busy
            ? (isNew ? t('compose.sending') : t('conversation.sending'))
            : (isNew ? t('compose.submit') : t('conversation.send'))}
        </button>
      </div>
      {hint && <p className="text-2xs text-ink-4 px-1">{hint}</p>}
    </form>
  )
}
