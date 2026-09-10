import type { Review } from '@/types'
import type { ReviewRepository } from '../types'
import { simulateLatency } from './latency'
import { REVIEWS, QUESTIONS } from '@/config/demoData'

const submitted: Review[] = []
let reviewSeq = 0

export class MockReviewRepository implements ReviewRepository {
  async listForProduct(productId: string, sort: 'newest' | 'highest' | 'lowest' | 'helpful' = 'newest'): Promise<Review[]> {
    await simulateLatency()
    const all = [...REVIEWS, ...submitted].filter(r => r.productId === productId)
    switch (sort) {
      case 'highest': return all.sort((a, b) => b.rating - a.rating)
      case 'lowest': return all.sort((a, b) => a.rating - b.rating)
      case 'helpful': return all.sort((a, b) => b.helpfulCount - a.helpfulCount)
      default: return all.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
    }
  }

  async submit(review: Omit<Review, 'id' | 'helpfulCount' | 'notHelpfulCount' | 'createdAt'>): Promise<Review> {
    await simulateLatency()
    reviewSeq += 1
    const full: Review = { ...review, id: `submitted-${reviewSeq}`, helpfulCount: 0, notHelpfulCount: 0, createdAt: new Date().toISOString() }
    submitted.push(full)
    return full
  }

  async markHelpful(reviewId: string, helpful: boolean): Promise<void> {
    await simulateLatency()
    const review = [...REVIEWS, ...submitted].find(r => r.id === reviewId)
    if (!review) return
    if (helpful) review.helpfulCount += 1
    else review.notHelpfulCount += 1
  }

  async questionsForProduct(productId: string) {
    await simulateLatency()
    return QUESTIONS.filter(q => q.productId === productId)
  }
}
