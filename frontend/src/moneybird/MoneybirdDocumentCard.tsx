import { ExternalLink, Send } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useMoneybirdDocument } from './hooks/useMoneybirdDocument'
import type { MoneybirdDocument, MoneybirdSourceType } from './types'

const DOCUMENT_PATH: Record<MoneybirdSourceType, string> = { quote: 'estimates', sales_order: 'external_sales_invoices' }

function moneybirdURL(administrationId: string, doc: MoneybirdDocument): string {
  return `https://moneybird.com/${administrationId}/${DOCUMENT_PATH[doc.source_type]}/${doc.moneybird_id}`
}

interface Props {
  sourceType: MoneybirdSourceType
  sourceId: string
  // For an order: whether it is paid -- a booked, unpaid order has nothing
  // more to send until it is.
  orderPaid?: boolean
  // For an order: its payment method; "" for a manual order without one,
  // whose payment is booked only if the shop mapped orders without a method.
  orderPaymentMethod?: string
  // Why the button is off right now (e.g. a quote with unsaved changes --
  // sending would send the saved version).
  blockedReason?: string
}

// Where a quote or sales order stands in Moneybird, with the button to
// send it. Nothing at all for a shop without Moneybird.
export function MoneybirdDocumentCard({ sourceType, sourceId, orderPaid = false, orderPaymentMethod, blockedReason }: Props) {
  const { token } = useAuth()
  const { t } = useTranslation('integrations')
  const mb = useMoneybirdDocument(token ?? '', sourceType, sourceId)
  if (!mb.connected || !mb.state) return null

  const doc = mb.state.document
  const booked = doc?.status === 'synced'
  const noMethod = orderPaymentMethod === ''
  const done = sourceType === 'sales_order' && booked && (doc.payment_registered || !orderPaid || noMethod)

  let status: string
  if (!doc) status = t('moneybird.card.notSent')
  else if (doc.status === 'failed') status = t('moneybird.card.failed', { error: doc.error })
  else if (sourceType === 'quote') status = t('moneybird.card.estimateSent')
  else if (doc.payment_registered) status = t('moneybird.card.bookedPaid')
  else status = orderPaid && noMethod ? t('moneybird.card.bookedNoMethod') : t('moneybird.card.booked')

  let action = t('moneybird.card.send')
  if (doc?.status === 'failed') action = t('moneybird.card.retry')
  else if (booked && sourceType === 'quote') action = t('moneybird.card.update')
  else if (booked) action = t('moneybird.card.registerPayment')

  return (
    <section aria-label={t('moneybird.name')} className="rounded-xl border border-line bg-raised p-4 space-y-2">
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-xs font-semibold text-ink-2">{t('moneybird.name')}</h3>
        {doc?.moneybird_id && (
          <a href={moneybirdURL(mb.state.administration_id, doc)} target="_blank" rel="noreferrer"
            className="inline-flex items-center gap-1 text-2xs text-accent hover:underline">
            {t('moneybird.card.open')} <ExternalLink size={11} aria-hidden />
          </a>
        )}
      </div>
      <p className={doc?.status === 'failed' ? 'text-xs text-danger' : 'text-xs text-ink-3'}>{status}</p>
      {mb.state.auto_sync && <p className="text-2xs text-ink-4">{t('moneybird.card.autoSyncOn')}</p>}
      {mb.sendError && <p role="alert" className="text-xs text-danger">{mb.sendError}</p>}
      {!done && (
        <button type="button" onClick={() => void mb.send()} disabled={mb.sending || Boolean(blockedReason)} title={blockedReason}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50">
          <Send size={12} aria-hidden /> {mb.sending ? t('moneybird.card.sending') : action}
        </button>
      )}
    </section>
  )
}
