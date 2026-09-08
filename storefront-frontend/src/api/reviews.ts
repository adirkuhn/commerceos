import { request } from './client'
import type { Review, ReviewList, SubmitReviewInput } from '../types'

export function listReviews(productId: string): Promise<ReviewList> {
  return request<ReviewList>(`/products/${encodeURIComponent(productId)}/reviews`)
}

export function submitReview(productId: string, input: SubmitReviewInput): Promise<Review> {
  return request<Review>(`/products/${encodeURIComponent(productId)}/reviews`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}
