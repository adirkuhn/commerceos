import type { ComponentDefinition } from '../types'

export interface ProductGridProps {
  source: 'all' | 'category' | 'products'
  category_id: string
  product_ids: string[]
  limit: number
  columns: number
  show_price: boolean
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
      options: [{ value: 'all', label: 'All products' }, { value: 'category', label: 'A category' }, { value: 'products', label: 'Products I choose' }],
    },
    category_id: { kind: 'category', label: 'Category' },
    product_ids: { kind: 'products', label: 'Products', max: 48 },
    limit: { kind: 'number', label: 'How many', min: 1, max: 48 },
    columns: { kind: 'select', label: 'Columns', options: [{ value: 0, label: 'Theme default' }, ...[2, 3, 4, 5, 6].map(n => ({ value: n, label: String(n) }))] },
    show_price: { kind: 'toggle', label: 'Show prices' },
  },
  defaults: { source: 'all', category_id: '', product_ids: [], limit: 8, columns: 0, show_price: true },
  showField: {
    category_id: p => p.source === 'category',
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
