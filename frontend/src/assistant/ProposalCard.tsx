import clsx from 'clsx'
import { CheckCircle2, Loader2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { quotePath, salesOrderPath } from './format'
import { ConversionPreview, QuotePreview } from './ProposalPreviews'
import type { AssistantProposal } from './types'

interface Props {
  proposal: AssistantProposal
  // The request is waiting on this decision -- only then can it be made.
  actionable: boolean
  busy: boolean
  error?: string
  onConfirm: (warehouseId?: string) => void
  onReject: () => void
}

// A change the assistant proposed: what it would create, and Confirm /
// Reject while it's pending. Nothing is written before Confirm.
export function ProposalCard({ proposal: p, actionable, busy, error, onConfirm, onReject }: Props) {
  const { t } = useTranslation('assistant')
  const [warehouseId, setWarehouseId] = useState(p.conversion?.warehouse_id ?? '')
  const pending = p.status === 'pending' && actionable
  const title = p.kind === 'convert_quote'
    ? t('proposal.convertTitle', { ref: p.conversion?.quote_ref ?? '' })
    : t('proposal.quoteTitle')

  return (
    <li
      data-testid={`assistant-proposal-${p.id}`}
      className={clsx('flex flex-col gap-3 rounded-xl border px-3 py-3 max-w-full',
        p.status === 'pending' ? 'border-accent/40 bg-surface' : 'border-line bg-sunken')}
    >
      <p className="text-2xs text-ink-4 caps-label">{title}</p>

      {p.quote && <QuotePreview preview={p.quote} />}
      {p.conversion && (
        <ConversionPreview preview={p.conversion} warehouseId={warehouseId} onWarehouse={setWarehouseId} editable={pending && !busy} />
      )}

      {p.last_error && p.status === 'pending' && (
        <p className="text-xs text-danger">
          <span className="text-ink-4">{t('proposal.lastError')}: </span>{p.last_error}
        </p>
      )}
      {error && <p className="text-xs text-danger">{error}</p>}

      <Outcome proposal={p} />

      {pending && (
        <div className="flex gap-2">
          <button
            type="button" disabled={busy} onClick={() => onConfirm(p.conversion ? warehouseId : undefined)}
            className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50"
          >
            {busy ? t('proposal.confirming') : t('proposal.confirm')}
          </button>
          <button
            type="button" disabled={busy} onClick={onReject}
            className="px-3 py-1.5 rounded-lg border border-line text-xs text-ink-2 hover:text-danger disabled:opacity-50"
          >
            {t('proposal.reject')}
          </button>
        </div>
      )}
    </li>
  )
}

function Outcome({ proposal: p }: { proposal: AssistantProposal }) {
  const { t } = useTranslation('assistant')
  switch (p.status) {
    case 'confirmed': {
      const ref = p.result_ref ?? ''
      const label = p.kind === 'convert_quote' ? t('proposal.createdOrder', { ref }) : t('proposal.createdQuote', { ref })
      const to = p.result_id ? (p.kind === 'convert_quote' ? salesOrderPath(p.result_id) : quotePath(p.result_id)) : null
      return (
        <p className="flex items-center gap-1.5 text-xs text-success">
          <CheckCircle2 size={12} aria-hidden />
          {to ? <Link to={to} className="hover:underline">{label}</Link> : label}
        </p>
      )
    }
    case 'executing':
      return <p className="flex items-center gap-1.5 text-xs text-ink-3"><Loader2 size={12} className="animate-spin" aria-hidden />{t('proposal.running')}</p>
    case 'rejected':
      return <p className="text-xs text-ink-4">{t('proposal.rejected')}</p>
    case 'superseded':
      return <p className="text-xs text-ink-4">{t('proposal.superseded')}</p>
    default:
      return null
  }
}
