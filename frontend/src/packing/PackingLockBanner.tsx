import { Lock } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatLockAge } from './lockDisplay'

interface PackingLockBannerProps {
  holderName: string
  lockedSince: string | null | undefined
  canTakeOver: boolean
  onTakeOver: () => void
}

// Someone else holds this session's lock -- it's view only here.
export function PackingLockBanner({ holderName, lockedSince, canTakeOver, onTakeOver }: PackingLockBannerProps) {
  const { t } = useTranslation('packing')
  return (
    <div role="status" className="flex items-center gap-2 px-6 py-2 border-b border-line bg-warning/10 text-xs text-ink-2">
      <Lock size={12} className="shrink-0" />
      <span className="flex-1">
        {t('lock.message', { name: holderName, since: lockedSince ? ` · ${formatLockAge(lockedSince)}` : '' })}
      </span>
      {canTakeOver && (
        <button type="button" onClick={onTakeOver} className="px-2 py-0.5 rounded-md border border-line bg-raised font-medium hover:bg-hover">
          {t('lock.takeOver')}
        </button>
      )}
    </div>
  )
}
