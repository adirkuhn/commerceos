import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { StockShortfall, StockShortfallResolution } from './types'
import { Dialog } from '../components/common/Dialog'

interface StockShortfallDialogProps {
  shortfalls: StockShortfall[]
  submitting: boolean
  onConfirm: (resolutions: Record<string, StockShortfallResolution>) => void
  onCancel: () => void
}

// Shown when finishing a packing (or legacy picking) session hits a
// substitute/freebie/new-item whose SKU doesn't have enough tracked stock
// (see backend OrderChangeApplier.CheckStockShortfalls) -- staff decide, per item, whether to
// ship it anyway (fixes the inventory record, reduces stock for real) or
// ship without it (the line just doesn't go out, same as any other unpicked
// line).
export function StockShortfallDialog({ shortfalls, submitting, onConfirm, onCancel }: StockShortfallDialogProps) {
  const { t } = useTranslation(['scanner', 'common'])
  const [choices, setChoices] = useState<Record<string, StockShortfallResolution>>(
    () => Object.fromEntries(shortfalls.map(s => [s.scan_id, 'with_item' as StockShortfallResolution])),
  )

  return (
    <Dialog onClose={onCancel} dismissible={!submitting} className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onCancel} />
      <div className="relative w-full max-w-md bg-raised rounded-xl shadow-2xl border border-line p-5">
        <h3 className="text-sm font-semibold text-ink mb-1">{t('shortfall.title')}</h3>
        <p className="text-xs text-ink-3 mb-4">
          {t('shortfall.intro')}
        </p>
        <div className="space-y-3 mb-4">
          {shortfalls.map(s => (
            <div key={s.scan_id} className="rounded-lg border border-line p-3">
              <p className="text-xs font-medium text-ink">{s.name}</p>
              <p className="text-xs text-ink-4 mb-2">
                {t('shortfall.needed', { needed: s.needed, short: s.short })}
              </p>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setChoices(prev => ({ ...prev, [s.scan_id]: 'with_item' }))}
                  aria-pressed={choices[s.scan_id] === 'with_item'}
                  className={`flex-1 px-2 py-2.5 rounded-lg text-xs font-medium transition-colors ${choices[s.scan_id] === 'with_item' ? 'bg-accent-solid text-white' : 'bg-fill text-ink-2'}`}
                >
                  {t('shortfall.send')}
                </button>
                <button
                  type="button"
                  onClick={() => setChoices(prev => ({ ...prev, [s.scan_id]: 'without_item' }))}
                  aria-pressed={choices[s.scan_id] === 'without_item'}
                  className={`flex-1 px-2 py-2.5 rounded-lg text-xs font-medium transition-colors ${choices[s.scan_id] === 'without_item' ? 'bg-accent-solid text-white' : 'bg-fill text-ink-2'}`}
                >
                  {t('shortfall.skip')}
                </button>
              </div>
            </div>
          ))}
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onCancel} disabled={submitting}
            className="px-4 py-2 rounded-lg bg-fill text-ink-2 text-xs font-medium disabled:opacity-50 transition-colors"
          >
            {t('common:actions.cancel')}
          </button>
          <button
            type="button" onClick={() => onConfirm(choices)} disabled={submitting}
            className="flex-1 px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {submitting ? t('review.completing') : t('review.confirm')}
          </button>
        </div>
      </div>
    </Dialog>
  )
}
