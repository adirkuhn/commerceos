import { ShoppingCart } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { useAddCartItem } from '@/features/cart/hooks'
import type { Product, ProductVariant } from '@/types'

export function AddToCartButton({
  product, variant, quantity = 1, disabled, size = 'default', label = 'Add to cart',
}: { product: Product; variant?: ProductVariant; quantity?: number; disabled?: boolean; size?: 'sm' | 'default' | 'lg'; label?: string }) {
  const addItem = useAddCartItem()
  const outOfStock = variant ? variant.stock <= 0 : product.stockStatus === 'out_of_stock'

  return (
    <Button
      type="button" size={size}
      disabled={disabled || outOfStock || addItem.isPending}
      onClick={() => addItem.mutate({
        productId: product.id,
        variantId: variant?.id,
        name: product.name,
        variantLabel: variant?.options.map(o => o.value).join(' / '),
        imageUrl: variant?.imageUrl ?? product.images[0]?.url,
        sku: variant?.sku ?? product.sku,
        price: variant?.price ?? product.price,
        quantity,
      })}
    >
      <ShoppingCart className="mr-2 h-4 w-4" />
      {outOfStock ? 'Out of stock' : addItem.isPending ? 'Adding…' : label}
    </Button>
  )
}
