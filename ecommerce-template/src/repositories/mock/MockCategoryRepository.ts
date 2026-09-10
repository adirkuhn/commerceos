import { CATEGORIES, BRANDS } from '@/config/demoData'
import type { CategoryRepository } from '../types'
import { simulateLatency } from './latency'

export class MockCategoryRepository implements CategoryRepository {
  async list() {
    await simulateLatency()
    return CATEGORIES
  }

  async getBySlug(slug: string) {
    await simulateLatency()
    return CATEGORIES.find(c => c.slug === slug) ?? null
  }

  async brands() {
    await simulateLatency()
    return BRANDS
  }
}
