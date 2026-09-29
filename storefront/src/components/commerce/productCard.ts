import type { ComponentDefinition } from '../types'

export interface ProductCardProps {
  product_id: string
  show_price: boolean
  show_add_to_cart: boolean
  show_stock: boolean
  show_discount: boolean
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
    show_discount: { kind: 'toggle', label: 'Show discount badge' },
    show_stock: { kind: 'toggle', label: 'Show stock status' },
    show_add_to_cart: { kind: 'toggle', label: 'Show add to cart' },
  },
  defaults: { product_id: '', show_price: true, show_add_to_cart: false, show_stock: false, show_discount: true },
  container: false,
  allowedChildren: [],
  allowedParents: [],
  dataContext: '',
  requiredIn: [],
  styleGroups: ['spacing', 'border', 'colors', 'layout', 'visibility'],
  editor: { removable: true, copyable: true, draggable: true },
}
