import type { ComponentDefinition } from '../types'

// Components for collection and search pages (data context "collection"),
// plus breadcrumbs, which work on any page.

const base = {
  container: false, allowedChildren: [], allowedParents: [], dataContext: 'collection' as const, requiredIn: [],
  editor: { removable: true, copyable: true, draggable: true },
}

export interface CollectionHeaderProps { show_image: boolean; show_description: boolean }
export const collectionHeader: ComponentDefinition<CollectionHeaderProps> = {
  ...base, type: 'collection-header', name: 'Collection title', category: 'commerce', icon: 'heading',
  description: 'The collection\'s name, picture and description (or the search term).',
  props: { show_image: { kind: 'toggle', label: 'Show picture' }, show_description: { kind: 'toggle', label: 'Show description' } },
  defaults: { show_image: true, show_description: true },
  styleGroups: ['typography', 'colors', 'spacing', 'visibility'],
}

export interface CollectionProductsProps { per_page: number; columns: number; show_price: boolean; show_add_to_cart: boolean; show_stock: boolean; show_discount: boolean }
export const collectionProducts: ComponentDefinition<CollectionProductsProps> = {
  ...base, type: 'collection-products', name: 'Collection products', category: 'commerce', icon: 'grid',
  description: 'The products in this collection or search, one page at a time.',
  props: {
    per_page: { kind: 'number', label: 'Products per page', min: 4, max: 48 },
    columns: { kind: 'select', label: 'Columns', options: [{ value: 0, label: 'Theme default' }, ...[2, 3, 4, 5, 6].map(n => ({ value: n, label: String(n) }))] },
    show_price: { kind: 'toggle', label: 'Show prices' },
    show_discount: { kind: 'toggle', label: 'Show discount badges' },
    show_stock: { kind: 'toggle', label: 'Show stock status' },
    show_add_to_cart: { kind: 'toggle', label: 'Show add to cart' },
  },
  defaults: { per_page: 24, columns: 0, show_price: true, show_add_to_cart: false, show_stock: false, show_discount: true },
  requiredIn: ['collection', 'search'],
  editor: { removable: false, copyable: false, draggable: true },
  styleGroups: ['spacing', 'visibility'],
}

export interface CollectionFiltersProps { title: string }
export const collectionFilters: ComponentDefinition<CollectionFiltersProps> = {
  ...base, type: 'collection-filters', name: 'Filters', category: 'commerce', icon: 'filter',
  description: 'Lets shoppers narrow the products by category.',
  props: { title: { kind: 'text', label: 'Title', maxLength: 60 } },
  defaults: { title: 'Filter' },
  styleGroups: ['spacing', 'visibility'],
}

export type PaginationProps = Record<string, never>
export const pagination: ComponentDefinition<PaginationProps> = {
  ...base, type: 'pagination', name: 'Pagination', category: 'commerce', icon: 'pages',
  description: 'Links to the next and previous pages of products.',
  props: {}, defaults: {},
  styleGroups: ['spacing', 'visibility'],
}

export interface BreadcrumbsProps { home_label: string }
export const breadcrumbs: ComponentDefinition<BreadcrumbsProps> = {
  ...base, dataContext: '', type: 'breadcrumbs', name: 'Breadcrumbs', category: 'navigation', icon: 'chevrons',
  description: 'Shows where the shopper is: Home › Collection › Product.',
  props: { home_label: { kind: 'text', label: 'Home label', maxLength: 40 } },
  defaults: { home_label: '' },
  styleGroups: ['typography', 'spacing', 'visibility'],
}
