import { useAuth } from '../auth/AuthContext'
import { useCustomerWhatsAppMessages } from './hooks/useCustomerWhatsAppMessages'

// Every WhatsApp message (inbound and the bot's outbound reply) recorded
// against this customer -- see backend/domain/whatsapp.Message. Read-only:
// there's no compose-a-message flow here, this is a log, not an inbox.
export function WhatsAppConversationPanel({ customerId }: { customerId: string }) {
  const { token } = useAuth()
  const { messages, loading, error } = useCustomerWhatsAppMessages(token ?? '', customerId)

  if (!loading && messages.length === 0 && !error) return null

  return (
    <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-2 text-xs">
      <p className="font-medium dark:text-gray-300 text-gray-700">WhatsApp conversation</p>
      {error && <p className="text-red-400">{error}</p>}
      {loading ? (
        <p className="dark:text-gray-500 text-gray-400">Loading…</p>
      ) : (
        <div className="space-y-2 max-h-80 overflow-y-auto">
          {messages.map(m => (
            <div key={m.id} className={`flex ${m.direction === 'outbound' ? 'justify-end' : 'justify-start'}`}>
              <div
                className={
                  m.direction === 'outbound'
                    ? 'max-w-[80%] rounded-lg px-2.5 py-1.5 bg-blue-600 text-white'
                    : 'max-w-[80%] rounded-lg px-2.5 py-1.5 dark:bg-[#1F2937] bg-gray-100 dark:text-gray-200 text-gray-800'
                }
              >
                <p className="whitespace-pre-wrap">{m.body}</p>
                <p className={`text-[10px] mt-0.5 ${m.direction === 'outbound' ? 'text-blue-100' : 'dark:text-gray-500 text-gray-400'}`}>
                  {new Date(m.created_at).toLocaleString()}
                </p>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
