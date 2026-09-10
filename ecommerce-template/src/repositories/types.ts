// Repository interfaces -- the seam between features/ and any real data
// source. Every method returns a Promise so a MockProductRepository and a
// future ApiProductRepository are interchangeable; see repositories/index.ts
// for the swap point. Components and hooks depend on these interfaces,
// never on a concrete Mock/Api class directly.
import type {
  Product, Category, Brand, Cart, CartItem, Order, Customer, Address, Review, Question,
  Coupon, FlashSale, SearchFilters, SearchResult, SearchSuggestion, CmsPage, BlogPost, FaqItem,
} from '@/types'

export interface ProductListParams {
  categoryId?: string
  brandId?: string
  tag?: string
  filter?: 'featured' | 'new_arrivals' | 'best_sellers' | 'recommended'
  excludeProductId?: string
  ids?: string[]
  page?: number
  perPage?: number
  sort?: SearchFilters['sort']
  minPrice?: number
  maxPrice?: number
  minRating?: number
  inStockOnly?: boolean
  onSaleOnly?: boolean
}

export interface ProductListResult {
  items: Product[]
  total: number
}

export interface ProductRepository {
  list(params: ProductListParams): Promise<ProductListResult>
  getBySlug(slug: string): Promise<Product | null>
  getById(id: string): Promise<Product | null>
  getByIds(ids: string[]): Promise<Product[]>
}

export interface CategoryRepository {
  list(): Promise<Category[]>
  getBySlug(slug: string): Promise<Category | null>
  brands(): Promise<Brand[]>
}

export interface CartRepository {
  get(cartId: string): Promise<Cart>
  addItem(cartId: string, item: Omit<CartItem, 'id'>): Promise<Cart>
  updateQuantity(cartId: string, lineId: string, quantity: number): Promise<Cart>
  removeItem(cartId: string, lineId: string): Promise<Cart>
  saveForLater(cartId: string, lineId: string): Promise<Cart>
  moveToCart(cartId: string, lineId: string): Promise<Cart>
  applyCoupon(cartId: string, code: string): Promise<Cart>
  removeCoupon(cartId: string, code: string): Promise<Cart>
  clear(cartId: string): Promise<Cart>
}

export interface CheckoutInput {
  email: string
  billingAddress: Address
  shippingAddress: Address
  shippingMethodId: Order['shippingMethod']
  paymentMethodId: Order['paymentMethod']
  customerId?: string
}

export interface CheckoutRepository {
  placeOrder(cart: Cart, input: CheckoutInput): Promise<Order>
}

export interface OrderRepository {
  listForCustomer(customerId: string): Promise<Order[]>
  getById(id: string): Promise<Order | null>
  cancel(id: string): Promise<Order>
  requestReturn(orderId: string, lineProductIds: string[], reason: string): Promise<Order>
  // Guest order tracking (/order-lookup) -- an order number alone isn't
  // enough to look up (it's not a secret), so this also requires the
  // email address the order was placed under, same as a real backend
  // would enforce server-side.
  findByOrderNumberAndEmail(orderNumber: string, email: string): Promise<Order | null>
}

export interface CustomerRepository {
  login(email: string, password: string): Promise<Customer>
  register(input: { firstName: string; lastName: string; email: string; password: string }): Promise<Customer>
  getById(id: string): Promise<Customer | null>
  update(id: string, patch: Partial<Customer>): Promise<Customer>
  addAddress(id: string, address: Omit<Address, 'id'>): Promise<Customer>
  updateAddress(id: string, address: Address): Promise<Customer>
  removeAddress(id: string, addressId: string): Promise<Customer>
}

export interface ReviewRepository {
  listForProduct(productId: string, sort?: 'newest' | 'highest' | 'lowest' | 'helpful'): Promise<Review[]>
  submit(review: Omit<Review, 'id' | 'helpfulCount' | 'notHelpfulCount' | 'createdAt'>): Promise<Review>
  markHelpful(reviewId: string, helpful: boolean): Promise<void>
  questionsForProduct(productId: string): Promise<Question[]>
}

export interface SearchRepository {
  search(filters: SearchFilters): Promise<SearchResult>
  suggestions(query: string): Promise<SearchSuggestion[]>
  popularSearches(): Promise<string[]>
}

export interface PromotionRepository {
  getCoupon(code: string): Promise<Coupon | null>
  activeFlashSales(): Promise<FlashSale[]>
}

export interface ContentRepository {
  getPage(slug: string): Promise<CmsPage | null>
  listBlogPosts(): Promise<BlogPost[]>
  getBlogPost(slug: string): Promise<BlogPost | null>
  faq(): Promise<FaqItem[]>
}
