import { Package } from 'lucide-react'
import { useProductMedia } from './hooks/useProductMedia'

// ThumbAppearance is the styling a screen family gives ProductThumb -- the
// admin UI and the dark scanner app size and colour it differently.
export interface ThumbAppearance {
  box: string
  iconSize: number
  iconClassName: string
}

export const ADMIN_THUMB: ThumbAppearance = { box: 'w-10 h-10 bg-hover', iconSize: 18, iconClassName: 'text-ink-4' }
export const SCANNER_THUMB: ThumbAppearance = { box: 'w-9 h-9 bg-night-700', iconSize: 16, iconClassName: 'text-gray-400' }

interface ProductThumbProps {
  token: string
  productId?: string | null
  appearance: ThumbAppearance
  title?: string
}

// ProductThumb is the small square product image on an order/booking line
// row -- the product's featured image, or a placeholder icon when the line
// isn't linked to a product yet or the product has no media.
export function ProductThumb({ token, productId, appearance, title }: ProductThumbProps) {
  const { media } = useProductMedia(token, productId ?? null)
  const featured = media[0]
  return (
    <div className={`${appearance.box} rounded-lg flex items-center justify-center shrink-0 overflow-hidden`} title={title}>
      {featured ? (
        <img src={featured.url} alt={featured.alt_text} className="w-full h-full object-cover" />
      ) : (
        <Package size={appearance.iconSize} className={appearance.iconClassName} />
      )}
    </div>
  )
}
