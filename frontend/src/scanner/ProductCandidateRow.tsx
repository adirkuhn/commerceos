import { Package } from 'lucide-react'
import { useProductMedia } from '../pms/hooks/useProductMedia'
import type { SKUCandidate } from '../pms/searchCandidates'

// Shared disambiguation row for every scan entry point's partial-match
// fallback (Inventory, Price Lookup, Booking) and Inventory's search-by-name
// list -- same look everywhere: thumbnail (falls back to a placeholder icon
// when the product has no media), product name, SKU code.
export function ProductCandidateRow({ token, candidate, onClick }: { token: string; candidate: SKUCandidate; onClick: () => void }) {
  const { media } = useProductMedia(token, candidate.productId)
  const featured = media[0]

  return (
    <button
      type="button"
      onClick={onClick}
      className="w-full flex items-center gap-3 p-2.5 rounded-lg border border-night-700 bg-night-800 text-left hover:border-accent transition-colors"
    >
      <div className="w-9 h-9 rounded-lg bg-night-700 flex items-center justify-center shrink-0 overflow-hidden">
        {featured ? (
          <img src={featured.url} alt={featured.alt_text} className="w-full h-full object-cover" />
        ) : (
          <Package size={16} className="text-gray-400" />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">{candidate.productName}</p>
        <p className="text-xs text-gray-400 truncate">{candidate.sku.sku_code}</p>
      </div>
    </button>
  )
}
