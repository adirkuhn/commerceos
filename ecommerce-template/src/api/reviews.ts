import { reviewRepository } from '@/repositories'
import type { Review } from '@/types'

export const listReviewsForProduct = (productId: string, sort?: 'newest' | 'highest' | 'lowest' | 'helpful') =>
  reviewRepository.listForProduct(productId, sort)
export const submitReview = (review: Omit<Review, 'id' | 'helpfulCount' | 'notHelpfulCount' | 'createdAt'>) =>
  reviewRepository.submit(review)
export const markReviewHelpful = (reviewId: string, helpful: boolean) => reviewRepository.markHelpful(reviewId, helpful)
export const listQuestionsForProduct = (productId: string) => reviewRepository.questionsForProduct(productId)
