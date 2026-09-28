import type { ComponentDefinition } from './types'
import { section } from './layout/section'
import { container } from './layout/container'
import { heading } from './content/heading'
import { text } from './content/text'
import { columns } from './layout/columns'
import { column } from './layout/column'
import { spacer } from './content/spacer'
import { image } from './content/image'
import { button } from './content/button'
import { productGrid } from './commerce/productGrid'
import { productCard } from './commerce/productCard'
import {
  addToCart, productDescription, productGallery, productPrice, productRecommendations, productReviews, productTitle,
  quantitySelector, variantSelector,
} from './commerce/productPage'
import { breadcrumbs, collectionFilters, collectionHeader, collectionProducts, pagination } from './commerce/listing'
import { cartItems, cartSummary, checkoutButton } from './cart/cart'
import { accountLink, cartLink, navigationMenu, searchBox, siteLogo } from './site/site'
import { banner, collectionList, hero, newsletterSignup } from './marketing/marketing'
import { accountOverview, loginForm, wishlistButton, wishlistItems } from './customer/customer'
import { checkoutForm, orderStatus } from './checkout/checkout'

// Every component the editor offers and the runtime renders. Adding one
// here (plus its renderer in renderers.tsx and its props schema in the
// backend contract) is all it takes.
const definitions = [
  section, container, columns, column, heading, text, image, button, spacer, productGrid, productCard,
  productGallery, productTitle, productPrice, variantSelector, quantitySelector, addToCart, productDescription,
  productReviews, productRecommendations,
  collectionHeader, collectionProducts, collectionFilters, pagination, breadcrumbs,
  cartItems, cartSummary, checkoutButton,
  siteLogo, navigationMenu, cartLink, searchBox, accountLink,
  hero, banner, collectionList, newsletterSignup,
  loginForm, accountOverview, wishlistItems, wishlistButton,
  checkoutForm, orderStatus,
] as unknown as ComponentDefinition[]

export class ComponentRegistry {
  private readonly byType: Map<string, ComponentDefinition>

  constructor(defs: ComponentDefinition[]) {
    this.byType = new Map()
    for (const d of defs) {
      if (this.byType.has(d.type)) throw new Error(`duplicate component type ${d.type}`)
      this.byType.set(d.type, d)
    }
  }

  get(type: string): ComponentDefinition | undefined {
    return this.byType.get(type)
  }

  all(): ComponentDefinition[] {
    return [...this.byType.values()]
  }

  /** Whether `child` may be placed directly inside `parentType`. */
  canPlace(child: string, parentType: string): boolean {
    const def = this.get(child)
    const parent = parentType === 'root' ? undefined : this.get(parentType)
    if (!def) return false
    if (parentType !== 'root' && !parent?.container) return false
    if (def.allowedParents.length && !def.allowedParents.includes(parentType)) return false
    if (parent?.allowedChildren.length && !parent.allowedChildren.includes(child)) return false
    return true
  }
}

export const registry = new ComponentRegistry(definitions)
