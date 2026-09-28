import type { ReactNode } from 'react'
import { Check } from 'lucide-react'
import { ProductThumb, SCANNER_THUMB } from '../pms/ProductThumb'
import type { PickingSessionLineItem } from './types'
import { SCAN_ACTIVITY_WORDS, type ScanActivity } from './scanSessionActivity'

// SoldExpiryNote styling on the scanner's always-dark line rows.
export const SCANNER_SOLD_EXPIRY_CLASS = 'text-xs text-gray-400 truncate'

interface ScanSessionLineRowProps {
  token: string
  lineItem: PickingSessionLineItem
  doneQty: number
  disabled: boolean
  onClick: () => void
  // Extra detail lines under the name/SKU code.
  children?: ReactNode
}

// One order line on a session screen -- tap to open its sheet.
export function ScanSessionLineRow({ token, lineItem: li, doneQty, disabled, onClick, children }: ScanSessionLineRowProps) {
  const done = doneQty >= li.quantity
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="w-full flex items-center gap-3 rounded-xl bg-night-800 border border-night-700 px-3.5 py-3 text-left hover:border-accent transition-colors"
    >
      <ProductThumb token={token} productId={li.product_id} appearance={SCANNER_THUMB} title={li.name} />
      {done && <Check size={16} className="text-green-400 shrink-0" />}
      <div className="flex-1 min-w-0">
        <p className="text-sm truncate">{li.name}</p>
        {li.sku_code && <p className="text-xs text-gray-400 truncate">{li.sku_code}</p>}
        {children}
      </div>
      <p className={`text-sm font-medium shrink-0 ${done ? 'text-green-400' : 'text-gray-300'}`}>
        {doneQty} / {li.quantity}
      </p>
    </button>
  )
}

interface MissingLineCardProps {
  activity: ScanActivity
  name: string
  expectedQty: number
  actualQty: number
}

// A line that wasn't fully picked/packed, on the review and done screens.
export function MissingLineCard({ activity, name, expectedQty, actualQty }: MissingLineCardProps) {
  return (
    <div className="rounded-lg bg-night-800 border border-night-700 px-3 py-2">
      <p className="text-sm text-gray-200">{name}</p>
      <p className="text-xs text-gray-400">expected {expectedQty} · {SCAN_ACTIVITY_WORDS[activity].done} {actualQty}</p>
    </div>
  )
}
