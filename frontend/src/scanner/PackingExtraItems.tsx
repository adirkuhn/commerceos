import { Gift, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { packScanLabel } from '../packing/packScanLabel'
import type { PackScan } from './types'

// Freebies and new items staged on this packing session, each removable.
export function PackingExtraItems({ extras, onRemove }: { extras: PackScan[]; onRemove: (scanId: string) => void }) {
  const { t } = useTranslation(['scanner', 'common'])
  const extraItem = t('extras.extraItem')
  if (extras.length === 0) return null
  return (
    <ul aria-label={t('extras.label')} className="space-y-2">
      {extras.map(s => (
        <li key={s.id} className="flex items-center gap-3 rounded-xl bg-night-800 border border-night-700 px-3.5 py-3">
          <Gift size={16} className="text-gray-400 shrink-0" />
          <p className="flex-1 min-w-0 text-sm truncate">{packScanLabel(s)}</p>
          <button aria-label={t('extras.remove', { name: s.reason || extraItem })}
            type="button" onClick={() => onRemove(s.id)} title={t('common:actions.remove')}
            className="shrink-0 w-11 h-11 -m-2 flex items-center justify-center rounded-lg text-gray-400 hover:text-red-400"
          >
            <X size={14} />
          </button>
        </li>
      ))}
    </ul>
  )
}
