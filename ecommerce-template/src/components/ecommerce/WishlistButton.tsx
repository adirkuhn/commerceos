import { Heart } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { useWishlistStore } from '@/stores/wishlistStore'
import { cn } from '@/lib/utils'

export function WishlistButton({ productId, className }: { productId: string; className?: string }) {
  const wishlisted = useWishlistStore(s => s.isWishlisted(productId))
  const toggle = useWishlistStore(s => s.toggle)

  return (
    <Button
      type="button" variant="outline" size="icon"
      aria-pressed={wishlisted}
      aria-label={wishlisted ? 'Remove from wishlist' : 'Add to wishlist'}
      onClick={() => toggle(productId)}
      className={className}
    >
      <Heart className={cn('h-4 w-4', wishlisted && 'fill-destructive text-destructive')} />
    </Button>
  )
}
