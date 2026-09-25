import { Package } from 'lucide-react'
import { useProductMedia } from '../pms/hooks/useProductMedia'

// Small square thumbnail for a booking line-item row -- falls back to a
// placeholder icon when the line isn't linked to a product yet, or the
// product has no media. Reuses the same media hook/data ProductCandidateRow
// already relies on for the scan-fallback picker.
export function LineItemThumb({ token, productId }: { token: string; productId?: string | null }) {
  const { media } = useProductMedia(token, productId ?? null)
  const featured = media[0]
  return (
    <div className="w-9 h-9 rounded-lg bg-night-700 flex items-center justify-center shrink-0 overflow-hidden">
      {featured ? (
        <img src={featured.url} alt={featured.alt_text} className="w-full h-full object-cover" />
      ) : (
        <Package size={16} className="text-gray-400" />
      )}
    </div>
  )
}
