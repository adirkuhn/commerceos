import type { ComponentDefinition } from '../types'

export interface ProductCardProps {
  product_id: string
  show_price: boolean
}

export const productCard: ComponentDefinition<ProductCardProps> = {
  type: 'product-card',
  name: 'Product card',
  category: 'commerce',
  icon: 'tag',
  description: 'One product you pick, with its picture and price.',
  props: {
    product_id: { kind: 'product', label: 'Product' },
    show_price: { kind: 'toggle', label: 'Show price' },
  },
  defaults: { product_id: '', show_price: true },
  container: false,
  allowedChildren: [],
  allowedParents: [],
  dataContext: '',
  requiredIn: [],
  styleGroups: ['spacing', 'border', 'colors', 'layout', 'visibility'],
  editor: { removable: true, copyable: true, draggable: true },
}
