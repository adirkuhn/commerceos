import { Gift, PackagePlus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { PickScan } from './types'

// Freebies and new items added to the order while picking -- read only here.
export function PickingAddedItems({ items }: { items: PickScan[] }) {
  const { t } = useTranslation('scanner')
  if (items.length === 0) return null
  return (
    <div className="pt-2 space-y-2">
      <p className="text-xs text-gray-400 px-1">{t('extras.addedDuringPicking')}</p>
      {items.map(s => (
        <div key={s.id} className="w-full flex items-center gap-3 rounded-xl bg-night-800/60 border border-night-700 px-3.5 py-3">
          {s.action_type === 'freebie' ? <Gift size={16} className="text-purple-400 shrink-0" /> : <PackagePlus size={16} className="text-accent shrink-0" />}
          <div className="flex-1 min-w-0">
            <p className="text-sm truncate">{s.reason || t('extras.newItem')}</p>
            <p className="text-xs text-gray-400 capitalize">{s.action_type === 'freebie' ? t('extras.freebie') : t('extras.newItem')}</p>
          </div>
          <p className="text-sm font-medium text-gray-300 shrink-0">×{s.quantity}</p>
        </div>
      ))}
    </div>
  )
}
