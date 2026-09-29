import type { ComponentDefinition } from '../types'

// Components that show the product a product page is about. They can only
// be placed on the product template (data context "product").

const base = {
  container: false, allowedChildren: [], allowedParents: [], dataContext: 'product' as const, requiredIn: [],
  editor: { removable: true, copyable: true, draggable: true },
}

export interface ProductGalleryProps { layout: 'thumbs_below' | 'stacked' }
export const productGallery: ComponentDefinition<ProductGalleryProps> = {
  ...base, type: 'product-gallery', name: 'Product images', category: 'commerce', icon: 'images',
  description: 'The product\'s pictures, with thumbnails to switch between them.',
  props: { layout: { kind: 'select', label: 'Layout', options: [{ value: 'thumbs_below', label: 'Thumbnails below' }, { value: 'stacked', label: 'All images stacked' }] } },
  defaults: { layout: 'thumbs_below' },
  styleGroups: ['spacing', 'border', 'visibility'],
}

export interface ProductTitleProps { level: number }
export const productTitle: ComponentDefinition<ProductTitleProps> = {
  ...base, type: 'product-title', name: 'Product name', category: 'commerce', icon: 'heading',
  description: 'The product\'s name as the page title.',
  props: { level: { kind: 'select', label: 'Level', options: [1, 2, 3].map(l => ({ value: l, label: `H${l}` })) } },
  defaults: { level: 1 },
  styleGroups: ['typography', 'colors', 'spacing', 'visibility'],
}

export interface ProductPriceProps { show_compare_at: boolean }
export const productPrice: ComponentDefinition<ProductPriceProps> = {
  ...base, type: 'product-price', name: 'Price', category: 'commerce', icon: 'tag',
  description: 'The chosen variant\'s price, with the old price crossed out during a sale.',
  props: { show_compare_at: { kind: 'toggle', label: 'Show the old price during a sale' } },
  defaults: { show_compare_at: true },
  styleGroups: ['typography', 'colors', 'spacing', 'visibility'],
}

export interface VariantSelectorProps { style: 'dropdown' | 'buttons' }
export const variantSelector: ComponentDefinition<VariantSelectorProps> = {
  ...base, type: 'variant-selector', name: 'Variant picker', category: 'commerce', icon: 'list',
  description: 'Lets shoppers choose a size, colour or other variant. Hidden for products without variants.',
  props: { style: { kind: 'select', label: 'Style', options: [{ value: 'buttons', label: 'Buttons' }, { value: 'dropdown', label: 'Dropdown' }] } },
  defaults: { style: 'buttons' },
  styleGroups: ['spacing', 'visibility'],
}

export interface QuantitySelectorProps { max: number }
export const quantitySelector: ComponentDefinition<QuantitySelectorProps> = {
  ...base, type: 'quantity-selector', name: 'Quantity', category: 'commerce', icon: 'hash',
  description: 'How many to add to the cart.',
  props: { max: { kind: 'number', label: 'Most per order', min: 1, max: 99 } },
  defaults: { max: 10 },
  styleGroups: ['spacing', 'visibility'],
}

export interface AddToCartProps { label: string; sold_out_label: string }
export const addToCart: ComponentDefinition<AddToCartProps> = {
  ...base, type: 'add-to-cart', name: 'Add to cart', category: 'commerce', icon: 'cart',
  description: 'Adds the chosen variant and quantity to the cart. Every product page needs one.',
  props: {
    label: { kind: 'text', label: 'Label', maxLength: 60 },
    sold_out_label: { kind: 'text', label: 'Label when sold out', maxLength: 60 },
  },
  defaults: { label: '', sold_out_label: '' },
  requiredIn: ['product'],
  editor: { removable: false, copyable: false, draggable: true },
  styleGroups: ['spacing', 'layout', 'visibility'],
}

export type ProductDescriptionProps = Record<string, never>
export const productDescription: ComponentDefinition<ProductDescriptionProps> = {
  ...base, type: 'product-description', name: 'Description', category: 'commerce', icon: 'text',
  description: 'The product description from your catalog.',
  props: {}, defaults: {},
  styleGroups: ['typography', 'colors', 'spacing', 'visibility'],
}

export interface ProductReviewsProps { title: string; limit: number }
export const productReviews: ComponentDefinition<ProductReviewsProps> = {
  ...base, type: 'product-reviews', name: 'Reviews', category: 'commerce', icon: 'star',
  description: 'The average rating and the latest reviews.',
  props: { title: { kind: 'text', label: 'Title', maxLength: 120 }, limit: { kind: 'number', label: 'How many reviews', min: 1, max: 50 } },
  defaults: { title: 'Reviews', limit: 10 },
  styleGroups: ['spacing', 'visibility'],
}

export interface ProductRecommendationsProps { title: string; limit: number }
export const productRecommendations: ComponentDefinition<ProductRecommendationsProps> = {
  ...base, type: 'product-recommendations', name: 'Recommendations', category: 'commerce', icon: 'sparkles',
  description: 'Products often bought together with this one.',
  props: { title: { kind: 'text', label: 'Title', maxLength: 120 }, limit: { kind: 'number', label: 'How many', min: 1, max: 12 } },
  defaults: { title: 'You may also like', limit: 4 },
  styleGroups: ['spacing', 'visibility'],
}
