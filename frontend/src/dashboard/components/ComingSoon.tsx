import { Construction } from 'lucide-react'
import clsx from 'clsx'

interface ComingSoonProps {
  label: string
  sub?: string
  className?: string
  height?: number
}

// Placeholder for dashboard widgets that need a tenant-wide reporting
// aggregate the backend doesn't expose yet (revenue, stock-flow, channel
// and product breakdowns) -- see useDashboardStats.
export function ComingSoon({ label, sub, className, height = 180 }: ComingSoonProps) {
  return (
    <div
      className={clsx(
        'flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed dark:border-[#2D3748] border-gray-200',
        className,
      )}
      style={{ minHeight: height }}
    >
      <Construction size={18} className="dark:text-gray-600 text-gray-300" />
      <p className="text-xs font-medium dark:text-gray-400 text-gray-500">Coming soon</p>
      <p className="text-[11px] dark:text-gray-500 text-gray-400">{sub ?? label}</p>
    </div>
  )
}
