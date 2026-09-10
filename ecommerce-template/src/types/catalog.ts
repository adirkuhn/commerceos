// Core catalog domain models. Prices are integer minor units (cents) --
// see lib/utils.ts formatMoney. Kept independent of any API/transport
// shape (see spec section 31): an ApiProductRepository would map its own
// wire DTOs onto these same types, components never see the difference.
import type { QuantityDiscountTier } from './promotions'

export type ProductKind = 'simple' | 'variable' | 'digital' | 'downloadable' | 'subscription' | 'bundle' | 'gift_card' | 'service'

export interface Money {
  amount: number // minor units
  currency: string
}

export interface Brand {
  id: string
  name: string
  slug: string
  logoUrl?: string
  description?: string
}

export interface Category {
  id: string
  name: string
  slug: string
  description?: string
  imageUrl?: string
  parentId?: string | null
}

export interface ProductAttribute {
  name: string
  value: string
}

export interface VariantOption {
  name: string // e.g. "Size", "Color"
  value: string // e.g. "M", "Charcoal"
}

export interface ProductVariant {
  id: string
  sku: string
  options: VariantOption[]
  price: number
  compareAtPrice?: number
  stock: number
  imageUrl?: string
  weightGrams?: number
}

export interface ProductImage {
  id: string
  url: string
  alt: string
}

export interface ProductDocument {
  id: string
  label: string
  url: string
  sizeLabel: string
}

export type StockStatus = 'in_stock' | 'low_stock' | 'out_of_stock' | 'backorder' | 'preorder'

export interface Product {
  id: string
  slug: string
  name: string
  sku: string
  kind: ProductKind
  brandId?: string
  categoryIds: string[]
  tags: string[]
  shortDescription: string
  description: string
  richContentHtml?: string
  images: ProductImage[]
  price: number
  compareAtPrice?: number
  currency: string
  taxIncluded: boolean
  attributes: ProductAttribute[]
  variantOptionNames: string[] // e.g. ["Size", "Color"] -- [] for a simple product
  variants: ProductVariant[]
  stock: number // for simple products; variable products sum/derive from variants
  stockStatus: StockStatus
  lowStockThreshold: number
  rating: number
  reviewCount: number
  isFeatured: boolean
  isNewArrival: boolean
  isBestSeller: boolean
  documents?: ProductDocument[]
  relatedProductIds?: string[]
  crossSellProductIds?: string[]
  upsellProductIds?: string[]
  frequentlyBoughtWithIds?: string[]
  weightGrams?: number
  createdAt: string
  // B2B wholesale terms -- absent for a consumer-only product. See
  // lib/b2bPricing.ts for how a company's price list further adjusts
  // `price` on top of these.
  minOrderQuantity?: number
  quantityBreaks?: QuantityDiscountTier[]
}

export function productStockLabel(product: Product): string {
  switch (product.stockStatus) {
    case 'out_of_stock':
      return 'Out of stock'
    case 'backorder':
      return 'Available on backorder'
    case 'preorder':
      return 'Available for preorder'
    case 'low_stock':
      return `Only ${product.stock} left`
    default:
      return 'In stock'
  }
}

export function productPriceRange(product: Product): { min: number; max: number } {
  if (product.variants.length === 0) return { min: product.price, max: product.price }
  const prices = product.variants.map(v => v.price)
  return { min: Math.min(...prices), max: Math.max(...prices) }
}
