import type { ComponentDefinition } from '../types'

export interface ProductGridProps {
  source: 'all' | 'category' | 'products' | 'collection'
  category_id: string
  collection_id: string
  product_ids: string[]
  limit: number
  columns: number
  show_price: boolean
  show_add_to_cart: boolean
  show_stock: boolean
  show_discount: boolean
  show_category: boolean
  add_button: 'full' | 'compact'
}

export const productGrid: ComponentDefinition<ProductGridProps> = {
  type: 'product-grid',
  name: 'Product grid',
  category: 'commerce',
  icon: 'grid',
  description: 'Products from your catalog: all, one category, or ones you pick.',
  props: {
    source: {
      kind: 'select', label: 'Show',
      options: [
        { value: 'all', label: 'All products' }, { value: 'collection', label: 'A collection' },
        { value: 'category', label: 'A category' }, { value: 'products', label: 'Products I choose' },
      ],
    },
    category_id: { kind: 'category', label: 'Category' },
    collection_id: { kind: 'collection', label: 'Collection' },
    product_ids: { kind: 'products', label: 'Products', max: 48 },
    limit: { kind: 'number', label: 'How many', min: 1, max: 48 },
    columns: { kind: 'select', label: 'Columns', options: [{ value: 0, label: 'Theme default' }, ...[2, 3, 4, 5, 6].map(n => ({ value: n, label: String(n) }))] },
    show_price: { kind: 'toggle', label: 'Show prices' },
    show_discount: { kind: 'toggle', label: 'Show discount badges' },
    show_stock: { kind: 'toggle', label: 'Show stock status' },
    show_add_to_cart: { kind: 'toggle', label: 'Show add to cart' },
    add_button: { kind: 'select', label: 'Add to cart button', options: [{ value: 'full', label: 'Wide, under the product' }, { value: 'compact', label: 'Small, beside the price' }] },
    show_category: { kind: 'toggle', label: 'Show category' },
  },
  defaults: { source: 'all', category_id: '', collection_id: '', product_ids: [], limit: 8, columns: 0, show_price: true, show_add_to_cart: false, show_stock: false, show_discount: true, show_category: false, add_button: 'full' },
  showField: {
    category_id: p => p.source === 'category',
    collection_id: p => p.source === 'collection',
    product_ids: p => p.source === 'products',
    limit: p => p.source !== 'products',
  },
  container: false,
  allowedChildren: [],
  allowedParents: [],
  dataContext: '',
  requiredIn: [],
  styleGroups: ['spacing', 'layout', 'visibility'],
  editor: { removable: true, copyable: true, draggable: true },
}
