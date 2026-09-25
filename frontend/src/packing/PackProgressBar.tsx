import clsx from 'clsx'

interface PackProgressBarProps {
  packedItems: number
  totalItems: number
  percent: number
  allPacked: boolean
}

export function PackProgressBar({ packedItems, totalItems, percent, allPacked }: PackProgressBarProps) {
  return (
    <div className="px-6 py-3 border-b border-line shrink-0">
      <div className="flex items-center justify-between mb-1.5">
        <span className="text-xs text-ink-3">Progress</span>
        <span className="text-xs font-medium text-ink">{packedItems} / {totalItems} items</span>
      </div>
      <div className="h-2 dark:bg-night-700 bg-gray-200 rounded-full overflow-hidden">
        <div
          className={clsx('h-full rounded-full transition-all duration-500', allPacked ? 'bg-green-500' : 'bg-accent-solid')}
          style={{ width: `${percent}%` }}
        />
      </div>
    </div>
  )
}
