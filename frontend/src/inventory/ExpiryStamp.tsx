import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useDateFormat } from '../format/DateFormatProvider'
import { isExpired, isExpiringSoon } from './inventoryFormat'

type ExpiryState = 'expired' | 'soon' | 'ok'

const STAMP: Record<ExpiryState, string> = {
  expired: 'border-danger text-danger bg-danger/10',
  soon: 'border-warning text-warning',
  ok: 'border-line text-ink-3',
}

// A lot's best-before date as a stamp: struck in red once past, amber within
// 30 days, quiet otherwise. The state is spelled out for screen readers.
export function ExpiryStamp({ date }: { date?: string | null }) {
  const { t } = useTranslation('inventory')
  const { formatDate } = useDateFormat()
  if (!date) return null
  const state: ExpiryState = isExpired(date) ? 'expired' : isExpiringSoon(date) ? 'soon' : 'ok'
  return (
    <span
      data-expiry={state}
      className={clsx('inline-flex items-center gap-1 rounded-sm border px-1.5 py-px text-2xs font-semibold tabular-nums whitespace-nowrap', STAMP[state])}
    >
      {state !== 'ok' && <span className="caps-label-wide">{t(state === 'expired' ? 'expiry.stampExpired' : 'expiry.stampSoon')}</span>}
      {formatDate(date)}
    </span>
  )
}
