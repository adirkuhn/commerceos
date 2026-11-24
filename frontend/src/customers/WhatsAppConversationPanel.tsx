import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useCustomerWhatsAppMessages } from './hooks/useCustomerWhatsAppMessages'
import { useDateFormat } from '../format/DateFormatProvider'

// Every WhatsApp message (inbound and the bot's outbound reply) recorded
// against this customer -- see backend/domain/whatsapp.Message. Read-only:
// there's no compose-a-message flow here, this is a log, not an inbox.
export function WhatsAppConversationPanel({ customerId }: { customerId: string }) {
  const { formatDateTime } = useDateFormat()
  const { token } = useAuth()
  const { t } = useTranslation(['customers', 'common'])
  const { messages, loading, error } = useCustomerWhatsAppMessages(token ?? '', customerId)

  if (!loading && messages.length === 0 && !error) return null

  return (
    <div className="rounded-xl border border-line p-4 space-y-2 text-xs">
      <p className="font-medium text-ink-2">{t('whatsapp.title')}</p>
      {error && <p className="text-danger">{error}</p>}
      {loading ? (
        <p className="text-ink-4">{t('common:status.loading')}</p>
      ) : (
        <div className="space-y-2 max-h-80 overflow-y-auto">
          {messages.map(m => (
            <div key={m.id} className={`flex ${m.direction === 'outbound' ? 'justify-end' : 'justify-start'}`}>
              <div
                className={
                  m.direction === 'outbound'
                    ? 'max-w-[80%] rounded-lg px-2.5 py-1.5 bg-accent-solid text-white'
                    : 'max-w-[80%] rounded-lg px-2.5 py-1.5 bg-fill text-ink'
                }
              >
                <p className="whitespace-pre-wrap">{m.body}</p>
                <p className={`text-2xs mt-0.5 ${m.direction === 'outbound' ? 'text-white/80' : 'text-ink-4'}`}>
                  {formatDateTime(m.created_at)}
                </p>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
