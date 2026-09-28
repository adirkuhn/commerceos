import { createContext, useContext, useMemo, useState, type ReactNode } from 'react'
import type { ProductDetail, ProductSKU, ProductVariant } from '../../api/storefront'

export interface ProductPage {
  product: ProductDetail
  variant: ProductVariant | undefined
  sku: ProductSKU | undefined
  selectVariant: (variantId: string) => void
  quantity: number
  setQuantity: (n: number) => void
}

const Ctx = createContext<ProductPage | null>(null)

const firstAvailable = (p: ProductDetail) =>
  p.variants.find(v => v.skus.some(s => s.stock_quantity > 0)) ?? p.variants[0]

/** The product a product page is about, plus what the shopper has chosen
 *  (variant, quantity) -- shared by the price, variant, quantity and
 *  add-to-cart components wherever they sit on the page. */
export function ProductProvider({ product, children }: { product: ProductDetail; children: ReactNode }) {
  const [variantId, setVariantId] = useState(() => firstAvailable(product)?.id)
  const [quantity, setQuantityState] = useState(1)
  const value = useMemo<ProductPage>(() => {
    const variant = product.variants.find(v => v.id === variantId) ?? product.variants[0]
    return {
      product, variant, sku: variant?.skus[0],
      selectVariant: setVariantId,
      quantity,
      setQuantity: n => setQuantityState(Math.max(1, Math.min(99, Math.round(n) || 1))),
    }
  }, [product, variantId, quantity])
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export const useProductPage = () => useContext(Ctx)
