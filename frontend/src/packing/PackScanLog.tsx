import { Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { PackScan } from '../scanner/types'
import { packScanLabel } from './packScanLabel'

interface PackScanLogProps {
  scans: PackScan[]
  onDelete: (scanId: string) => void
}

// Everything staged on this session so far, each removable.
export function PackScanLog({ scans, onDelete }: PackScanLogProps) {
  const { t } = useTranslation('packing')
  if (scans.length === 0) return null
  return (
    <div className="border-t border-line px-4 py-2 max-h-32 overflow-y-auto shrink-0">
      <p className="text-2xs font-medium text-ink-4 caps-label mb-1">{t('scanLog.title')}</p>
      {scans.map((s: PackScan) => (
        <div key={s.id} className="flex items-center justify-between py-0.5">
          <span className="text-2xs text-ink-3">{packScanLabel(s)}</span>
          <button type="button" onClick={() => onDelete(s.id)} aria-label={t('scanLog.remove')} className="text-ink-4 hover:text-danger">
            <Trash2 size={11} />
          </button>
        </div>
      ))}
    </div>
  )
}
