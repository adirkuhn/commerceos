import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import clsx from 'clsx'
import { Trash2 } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { useLocation } from 'react-router-dom'
import { idFromPath } from '../routes'
import { useStackNavigation } from '../routing/useStackNavigation'
import { CONVERSATIONS_PATH } from '../messagecenter/messageCenterPaths'
import { useWhatsAppConversations } from './hooks/useWhatsAppConversations'
import { useWhatsAppConversationMessages } from './hooks/useWhatsAppConversationMessages'
import type { WhatsAppConversation } from './types'
import { useConfirm } from '../components/common/useConfirm'
import { useDateFormat } from '../format/DateFormatProvider'
import { apiErrorText } from '../api/apiErrorText'

function conversationLabel(conv: WhatsAppConversation): string {
  return conv.customer_name || conv.phone_e164
}

// A "WhatsApp Web"-style admin view: every conversation this tenant's
// WhatsApp integration has ever seen (see backend/domain/whatsapp.Message),
// newest first, with the selected one's full thread on the right.
// Read-only -- there's no compose/reply box, replying happens through the
// bot or the outbound notification flows, not from here.
export default function WhatsAppConversationsPanel() {
  const { formatDate, formatDateTime } = useDateFormat()
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { t } = useTranslation(['messages', 'common'])
  const { conversations, loading, error, deleteConversation } = useWhatsAppConversations(token ?? '')
  // Selected thread is the URL (/message-center/conversations/<e164>).
  const location = useLocation()
  const { open, exitTo } = useStackNavigation()
  const selectedPhone = idFromPath(location.pathname, CONVERSATIONS_PATH)
  const setSelectedPhone = (phone: string | null) => {
    if (phone === null) exitTo(CONVERSATIONS_PATH)
    else if (phone !== selectedPhone) open(`${CONVERSATIONS_PATH}/${encodeURIComponent(phone)}`)
  }
  const { messages, loading: loadingMessages } = useWhatsAppConversationMessages(token ?? '', selectedPhone)
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState('')

  const selected = conversations.find(c => c.phone_e164 === selectedPhone) ?? null

  async function handleDelete() {
    if (!selected) return
    if (!(await confirm({ title: t('whatsapp.deleteTitle', { name: conversationLabel(selected) }), message: t('whatsapp.deleteMessage'), confirmLabel: t('whatsapp.deleteConfirm'), destructive: true }))) {
      return
    }
    setDeleting(true)
    setDeleteError('')
    try {
      await deleteConversation(selected.phone_e164)
      setSelectedPhone(null)
    } catch (err) {
      setDeleteError(apiErrorText(err) ?? t('whatsapp.deleteFailed'))
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="flex h-full overflow-hidden">
      <div className="w-72 shrink-0 border-r border-line flex flex-col overflow-hidden">
        <div className="px-4 py-3 border-b border-line shrink-0">
          <h2 className="text-sm font-semibold text-ink">{t('whatsapp.title')}</h2>
        </div>
        <div className="flex-1 overflow-y-auto">
          {error && <p className="text-xs text-danger p-3">{error}</p>}
          {loading && conversations.length === 0 ? (
            <p className="text-xs text-ink-4 p-3">{t('common:status.loading')}</p>
          ) : conversations.length === 0 ? (
            <p className="text-xs text-ink-4 p-3">{t('whatsapp.empty')}</p>
          ) : (
            conversations.map(conv => (
              <button
                key={conv.phone_e164}
                onClick={() => setSelectedPhone(conv.phone_e164)}
                className={clsx(
                  'w-full text-left px-4 py-2.5 border-b border-line transition-colors',
                  conv.phone_e164 === selectedPhone
                    ? 'bg-fill'
                    : 'hover:bg-hover/60',
                )}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs font-medium text-ink truncate">{conversationLabel(conv)}</span>
                  <span className="text-2xs text-ink-4 shrink-0">
                    {formatDate(conv.last_message_at)}
                  </span>
                </div>
                <p className="text-2xs text-ink-4 truncate mt-0.5">
                  {`${conv.last_direction === 'outbound' ? '↩ ' : ''}${conv.last_body}`}
                </p>
              </button>
            ))
          )}
        </div>
      </div>

      <div className="flex-1 flex flex-col overflow-hidden">
        {!selected ? (
          <div className="flex-1 flex items-center justify-center">
            <p className="text-xs text-ink-4">{t('whatsapp.select')}</p>
          </div>
        ) : (
          <>
            <div className="px-5 py-3 border-b border-line shrink-0 flex items-center justify-between gap-3">
              <div>
                <p className="text-sm font-semibold text-ink">{conversationLabel(selected)}</p>
                {selected.customer_name && (
                  <p className="text-2xs text-ink-4">{selected.phone_e164}</p>
                )}
              </div>
              <button
                onClick={handleDelete}
                disabled={deleting}
                title={t('whatsapp.deleteChat')}
                aria-label={t('whatsapp.deleteChat')}
                className="shrink-0 w-7 h-7 flex items-center justify-center rounded hover:bg-hover disabled:opacity-50 transition-colors"
              >
                <Trash2 size={14} className="text-ink-4 hover:text-danger" />
              </button>
            </div>
            {deleteError && <p className="text-xs text-danger px-5 pt-2">{deleteError}</p>}
            <div className="flex-1 overflow-y-auto p-5 space-y-2">
              {loadingMessages && messages.length === 0 ? (
                <p className="text-xs text-ink-4">{t('common:status.loading')}</p>
              ) : (
                [...messages].reverse().map(m => (
                  <div key={m.id} className={`flex ${m.direction === 'outbound' ? 'justify-end' : 'justify-start'}`}>
                    <div
                      className={
                        m.direction === 'outbound'
                          ? 'max-w-[70%] rounded-lg px-3 py-2 bg-accent-solid text-white'
                          : 'max-w-[70%] rounded-lg px-3 py-2 bg-fill text-ink'
                      }
                    >
                      <p className="text-xs whitespace-pre-wrap">{m.body}</p>
                      <p className={`text-2xs mt-0.5 ${m.direction === 'outbound' ? 'text-white/80' : 'text-ink-4'}`}>
                        {formatDateTime(m.created_at)}
                      </p>
                    </div>
                  </div>
                ))
              )}
            </div>
          </>
        )}
      </div>
      {confirmDialog}
    </div>
  )
}
