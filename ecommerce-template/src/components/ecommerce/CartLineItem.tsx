import { X, BookmarkPlus } from 'lucide-react'
import type { CartItem } from '@/types'
import { formatMoney } from '@/lib/utils'
import { QuantitySelector } from './QuantitySelector'
import { Button } from '@/components/ui/Button'
import { useUpdateCartItemQuantity, useRemoveCartItem, useSaveForLater, useMoveToCart } from '@/features/cart/hooks'

export function CartLineItem({ item, currency }: { item: CartItem; currency: string }) {
  const updateQuantity = useUpdateCartItemQuantity()
  const removeItem = useRemoveCartItem()
  const saveForLater = useSaveForLater()
  const moveToCart = useMoveToCart()

  return (
    <div className="flex gap-3 py-4">
      {item.imageUrl && <img src={item.imageUrl} alt="" className="h-20 w-20 shrink-0 rounded-[var(--radius)] object-cover" />}
      <div className="flex flex-1 flex-col justify-between">
        <div>
          <p className="text-sm font-medium">{item.name}</p>
          {item.variantLabel && <p className="text-xs text-muted-foreground">{item.variantLabel}</p>}
          <p className="text-xs text-muted-foreground">SKU {item.sku}</p>
        </div>
        <div className="mt-2 flex items-center justify-between gap-3">
          {item.savedForLater ? (
            <Button type="button" variant="outline" size="sm" onClick={() => moveToCart.mutate(item.id)}>Move to cart</Button>
          ) : (
            <QuantitySelector value={item.quantity} onChange={(q) => updateQuantity.mutate({ lineId: item.id, quantity: q })} />
          )}
          <p className="text-sm font-medium">{formatMoney(item.price * item.quantity, currency)}</p>
        </div>
      </div>
      <div className="flex flex-col items-end gap-1.5">
        <Button type="button" variant="ghost" size="icon" className="h-7 w-7" aria-label="Remove item" onClick={() => removeItem.mutate({ lineId: item.id, productId: item.productId })}>
          <X className="h-3.5 w-3.5" />
        </Button>
        {!item.savedForLater && (
          <Button type="button" variant="ghost" size="icon" className="h-7 w-7" aria-label="Save for later" onClick={() => saveForLater.mutate(item.id)}>
            <BookmarkPlus className="h-3.5 w-3.5" />
          </Button>
        )}
      </div>
    </div>
  )
}
