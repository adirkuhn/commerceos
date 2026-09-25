import { Package } from 'lucide-react'
import { useProductMedia } from '../pms/hooks/useProductMedia'

// Falls back to a placeholder icon when the line isn't linked to a product
// yet, or the product has no media -- same convention as the scanner app's
// own PickingScreen/BookingScreen thumbnails.
export function LineItemThumb({ token, productId }: { token: string; productId?: string | null }) {
  const { media } = useProductMedia(token, productId ?? null)
  const featured = media[0]
  return (
    <div className="w-10 h-10 rounded-lg bg-hover flex items-center justify-center shrink-0 overflow-hidden">
      {featured ? (
        <img src={featured.url} alt={featured.alt_text} className="w-full h-full object-cover" />
      ) : (
        <Package size={18} className="text-ink-4" />
      )}
    </div>
  )
}
