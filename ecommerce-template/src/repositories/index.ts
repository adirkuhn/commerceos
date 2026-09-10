// The swap point: every api/*.ts file imports its repository from here,
// never instantiates Mock*Repository directly. Today VITE_API_MODE is
// always "mock" (the default) -- once a real Go backend exists, add
// ApiProductRepository etc alongside the Mock ones and switch the export
// below on import.meta.env.VITE_API_MODE. Nothing outside this file
// needs to change.
import { MockProductRepository } from './mock/MockProductRepository'
import { MockCategoryRepository } from './mock/MockCategoryRepository'
import { MockCartRepository } from './mock/MockCartRepository'
import { MockCheckoutRepository } from './mock/MockCheckoutRepository'
import { MockOrderRepository } from './mock/MockOrderRepository'
import { MockCustomerRepository } from './mock/MockCustomerRepository'
import { MockReviewRepository } from './mock/MockReviewRepository'
import { MockSearchRepository } from './mock/MockSearchRepository'
import { MockPromotionRepository } from './mock/MockPromotionRepository'
import { MockContentRepository } from './mock/MockContentRepository'

const apiMode = import.meta.env.VITE_API_MODE ?? 'mock'
if (apiMode !== 'mock') {
  throw new Error(`VITE_API_MODE="${apiMode}" has no Api*Repository implementation yet -- add one alongside the Mock ones in src/repositories and wire it in here.`)
}

export const productRepository = new MockProductRepository()
export const categoryRepository = new MockCategoryRepository()
export const cartRepository = new MockCartRepository()
export const checkoutRepository = new MockCheckoutRepository()
export const orderRepository = new MockOrderRepository()
export const customerRepository = new MockCustomerRepository()
export const reviewRepository = new MockReviewRepository()
export const searchRepository = new MockSearchRepository()
export const promotionRepository = new MockPromotionRepository()
export const contentRepository = new MockContentRepository()

export * from './types'
