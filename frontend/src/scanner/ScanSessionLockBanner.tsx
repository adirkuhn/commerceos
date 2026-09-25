import { Lock } from 'lucide-react'
import { formatLockAge } from '../packing/lockDisplay'
import { SCAN_ACTIVITY_WORDS, type ScanActivity } from './scanSessionActivity'

interface ScanSessionLockBannerProps {
  activity: ScanActivity
  holderName: string
  lockedSince?: string | null
  canTakeOver: boolean
  onTakeOver: () => void
}

// Shown while someone else holds the session -- the screen is view only.
export function ScanSessionLockBanner({ activity, holderName, lockedSince, canTakeOver, onTakeOver }: ScanSessionLockBannerProps) {
  return (
    <div role="status" className="flex items-center gap-2 px-4 py-2 border-b border-night-700 bg-amber-500/10 text-sm text-amber-200 shrink-0">
      <Lock size={14} className="shrink-0" />
      <span className="flex-1">
        {holderName} is {SCAN_ACTIVITY_WORDS[activity].doing} this order{lockedSince ? ` · ${formatLockAge(lockedSince)}` : ''} -- view only.
      </span>
      {canTakeOver && (
        <button type="button" onClick={onTakeOver} className="min-h-11 px-3 -my-2 rounded-lg bg-night-800 border border-night-700 text-gray-100 font-medium">
          Take over
        </button>
      )}
    </div>
  )
}
