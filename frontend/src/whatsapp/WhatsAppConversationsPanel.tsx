import { useState } from 'react'
import clsx from 'clsx'
import { Trash2 } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { useWhatsAppConversations } from './hooks/useWhatsAppConversations'
import { useWhatsAppConversationMessages } from './hooks/useWhatsAppConversationMessages'
import type { WhatsAppConversation } from './types'

function conversationLabel(conv: WhatsAppConversation): string {
  return conv.customer_name || conv.phone_e164
}

// A "WhatsApp Web"-style admin view: every conversation this tenant's
// WhatsApp integration has ever seen (see backend/domain/whatsapp.Message),
// newest first, with the selected one's full thread on the right.
// Read-only -- there's no compose/reply box, replying happens through the
// bot or the outbound notification flows, not from here.
export default function WhatsAppConversationsPanel() {
  const { token } = useAuth()
  const { conversations, loading, error, deleteConversation } = useWhatsAppConversations(token ?? '')
  const [selectedPhone, setSelectedPhone] = useState<string | null>(null)
  const { messages, loading: loadingMessages } = useWhatsAppConversationMessages(token ?? '', selectedPhone)
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState('')

  const selected = conversations.find(c => c.phone_e164 === selectedPhone) ?? null

  async function handleDelete() {
    if (!selected) return
    if (!window.confirm(`Delete this conversation with ${conversationLabel(selected)}? This permanently removes its message history and cannot be undone.`)) {
      return
    }
    setDeleting(true)
    setDeleteError('')
    try {
      await deleteConversation(selected.phone_e164)
      setSelectedPhone(null)
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Failed to delete conversation')
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="flex h-full overflow-hidden">
      <div className="w-72 shrink-0 border-r dark:border-[#2D3748] border-gray-200 flex flex-col overflow-hidden">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
          <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Conversations</h2>
        </div>
        <div className="flex-1 overflow-y-auto">
          {error && <p className="text-xs text-red-400 p-3">{error}</p>}
          {loading && conversations.length === 0 ? (
            <p className="text-xs dark:text-gray-500 text-gray-400 p-3">Loading…</p>
          ) : conversations.length === 0 ? (
            <p className="text-xs dark:text-gray-500 text-gray-400 p-3">No WhatsApp conversations yet.</p>
          ) : (
            conversations.map(conv => (
              <button
                key={conv.phone_e164}
                onClick={() => setSelectedPhone(conv.phone_e164)}
                className={clsx(
                  'w-full text-left px-4 py-2.5 border-b dark:border-[#2D3748] border-gray-100 transition-colors',
                  conv.phone_e164 === selectedPhone
                    ? 'dark:bg-[#1F2937] bg-gray-100'
                    : 'hover:dark:bg-[#1F2937]/60 hover:bg-gray-50',
                )}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs font-medium dark:text-gray-100 text-gray-900 truncate">{conversationLabel(conv)}</span>
                  <span className="text-[10px] dark:text-gray-500 text-gray-400 shrink-0">
                    {new Date(conv.last_message_at).toLocaleDateString()}
                  </span>
                </div>
                <p className="text-[11px] dark:text-gray-500 text-gray-400 truncate mt-0.5">
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
            <p className="text-xs dark:text-gray-500 text-gray-400">Select a conversation</p>
          </div>
        ) : (
          <>
            <div className="px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0 flex items-center justify-between gap-3">
              <div>
                <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">{conversationLabel(selected)}</p>
                {selected.customer_name && (
                  <p className="text-[11px] dark:text-gray-500 text-gray-400">{selected.phone_e164}</p>
                )}
              </div>
              <button
                onClick={handleDelete}
                disabled={deleting}
                title="Delete chat"
                aria-label="Delete chat"
                className="shrink-0 w-7 h-7 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 disabled:opacity-50 transition-colors"
              >
                <Trash2 size={14} className="dark:text-gray-500 text-gray-400 hover:text-red-400" />
              </button>
            </div>
            {deleteError && <p className="text-xs text-red-400 px-5 pt-2">{deleteError}</p>}
            <div className="flex-1 overflow-y-auto p-5 space-y-2">
              {loadingMessages && messages.length === 0 ? (
                <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>
              ) : (
                [...messages].reverse().map(m => (
                  <div key={m.id} className={`flex ${m.direction === 'outbound' ? 'justify-end' : 'justify-start'}`}>
                    <div
                      className={
                        m.direction === 'outbound'
                          ? 'max-w-[70%] rounded-lg px-3 py-2 bg-blue-600 text-white'
                          : 'max-w-[70%] rounded-lg px-3 py-2 dark:bg-[#1F2937] bg-gray-100 dark:text-gray-200 text-gray-800'
                      }
                    >
                      <p className="text-xs whitespace-pre-wrap">{m.body}</p>
                      <p className={`text-[10px] mt-0.5 ${m.direction === 'outbound' ? 'text-blue-100' : 'dark:text-gray-500 text-gray-400'}`}>
                        {new Date(m.created_at).toLocaleString()}
                      </p>
                    </div>
                  </div>
                ))
              )}
            </div>
          </>
        )}
      </div>
    </div>
  )
}
