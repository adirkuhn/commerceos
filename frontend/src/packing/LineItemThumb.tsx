import { Package } from 'lucide-react'
import { useProductMedia } from '../pms/hooks/useProductMedia'

// Duplicated from Picking.tsx on purpose -- same small-helper convention
// used elsewhere in this codebase rather than a shared cross-screen import.
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
