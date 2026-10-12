import { CheckCircle, Trash2 } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import type { PickScan, PickingSessionLineItem } from '../scanner/types'
import { ProductThumb, ADMIN_THUMB } from '../pms/ProductThumb'
import { SoldExpiryNote } from '../oms/SoldExpiryNote'
import { actionLabel } from './pickingDisplay'
import type { PickProgress } from './pickingProgress'

interface PickProgressBarProps {
  progress: PickProgress
}

export function PickProgressBar({ progress }: PickProgressBarProps) {
  const { t } = useTranslation('picking')
  return (
    <div className="px-6 py-3 border-b border-line shrink-0">
      <div className="flex items-center justify-between mb-1.5">
        <span className="text-xs text-ink-3">{t('progress.label')}</span>
        <span className="text-xs font-medium text-ink">{t('progress.items', { picked: progress.pickedItems, total: progress.totalItems })}</span>
      </div>
      <div className="h-2 dark:bg-night-700 bg-gray-200 rounded-full overflow-hidden">
        <div
          className={clsx('h-full rounded-full transition-all duration-500', progress.allPicked ? 'bg-green-500' : 'bg-accent-solid')}
          style={{ width: `${progress.percent}%` }}
        />
      </div>
    </div>
  )
}

interface PickLineListProps {
  token: string
  lineItems: PickingSessionLineItem[]
  pickedByLineItem: Map<string, number>
  readOnly: boolean
  onOpenLine: (lineItem: PickingSessionLineItem, pickedSoFar: number) => void
}

// The order's lines, each with picked/ordered; tapping an unfinished one
// opens the pick sheet.
export function PickLineList({ token, lineItems, pickedByLineItem, readOnly, onOpenLine }: PickLineListProps) {
  const { t } = useTranslation('picking')
  return (
    <div className="flex-1 overflow-y-auto p-4 space-y-2">
      {lineItems.map(li => {
        const picked = pickedByLineItem.get(li.id) ?? 0
        const done = picked >= li.quantity
        const partial = picked > 0 && picked < li.quantity
        return (
          <div
            key={li.id}
            onClick={() => !done && !readOnly && onOpenLine(li, picked)}
            className={clsx(
              'flex items-center gap-4 p-3 rounded-xl border transition-all',
              li.sku_id && !done && 'cursor-pointer',
              done ? 'dark:bg-green-500/5 dark:border-green-500/20 border-green-200 bg-green-50' :
              partial ? 'dark:bg-amber-500/5 dark:border-amber-500/20 border-amber-200 bg-amber-50' :
              'bg-raised border-line',
            )}
          >
            {done ? (
              <div className="w-10 h-10 rounded-lg bg-green-500/20 flex items-center justify-center shrink-0">
                <CheckCircle size={18} className="text-success" />
              </div>
            ) : (
              <ProductThumb token={token} productId={li.product_id} appearance={ADMIN_THUMB} />
            )}
            <div className="flex-1 min-w-0">
              <p className={clsx('text-sm font-medium', done ? 'dark:text-green-300 text-success' : 'text-ink')}>{li.name}</p>
              <div className="flex items-center gap-2 mt-0.5">
                <span className="text-2xs font-mono text-ink-4">{li.sku_code}</span>
                {!li.sku_id && <span className="text-2xs text-warning">{t('line.unlinked')}</span>}
              </div>
              <SoldExpiryNote date={li.sold_expiry_date} />
            </div>
            <div className="text-right shrink-0">
              <p className={clsx('text-lg font-bold tabular-nums', done ? 'text-success' : partial ? 'text-warning' : 'text-ink')}>
                {picked}/{li.quantity}
              </p>
              <p className="text-2xs text-ink-4">{t('line.picked')}</p>
            </div>
          </div>
        )
      })}
    </div>
  )
}

interface PickScanLogProps {
  scans: PickScan[]
  onDelete: (scanId: string) => void
}

// The session's staged scans, each removable. Hidden while empty.
export function PickScanLog({ scans, onDelete }: PickScanLogProps) {
  const { t } = useTranslation('picking')
  if (scans.length === 0) return null
  return (
    <div className="border-t border-line px-4 py-2 max-h-32 overflow-y-auto shrink-0">
      <p className="text-2xs font-medium text-ink-4 caps-label mb-1">{t('scanLog.title')}</p>
      {scans.map(s => (
        <div key={s.id} className="flex items-center justify-between py-0.5">
          <span className="text-2xs text-ink-3">
            {t('scanLog.entry', { action: actionLabel(s.action_type), quantity: s.quantity })}
          </span>
          <button type="button" onClick={() => onDelete(s.id)} aria-label={t('scanLog.remove')} className="text-ink-4 hover:text-danger">
            <Trash2 size={11} />
          </button>
        </div>
      ))}
    </div>
  )
}
