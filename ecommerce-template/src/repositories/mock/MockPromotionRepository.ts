import type { PromotionRepository } from '../types'
import { simulateLatency } from './latency'
import { COUPONS, FLASH_SALES } from '@/config/demoData'

export class MockPromotionRepository implements PromotionRepository {
  async getCoupon(code: string) {
    await simulateLatency()
    return COUPONS.find(c => c.code === code.trim().toUpperCase()) ?? null
  }

  async activeFlashSales() {
    await simulateLatency()
    return FLASH_SALES.filter(f => new Date(f.endsAt).getTime() > Date.now())
  }
}
