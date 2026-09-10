import { Star } from 'lucide-react'
import { cn } from '@/lib/utils'

export function ProductRating({ rating, reviewCount, size = 14, showCount = true }: { rating: number; reviewCount: number; size?: number; showCount?: boolean }) {
  if (reviewCount === 0) {
    return <span className="text-xs text-muted-foreground">No reviews yet</span>
  }
  return (
    <div className="flex items-center gap-1" role="img" aria-label={`Rated ${rating} out of 5 stars, ${reviewCount} reviews`}>
      <div className="flex" aria-hidden="true">
        {Array.from({ length: 5 }, (_, i) => (
          <Star
            key={i}
            width={size}
            height={size}
            className={cn(i < Math.round(rating) ? 'fill-amber-400 text-amber-400' : 'fill-none text-muted-foreground')}
          />
        ))}
      </div>
      {showCount && <span className="text-xs text-muted-foreground">({reviewCount})</span>}
    </div>
  )
}
