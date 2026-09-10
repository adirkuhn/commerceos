import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { listReviewsForProduct, submitReview, markReviewHelpful, listQuestionsForProduct } from '@/api/reviews'
import type { Review, RatingDistribution } from '@/types'

export function useProductReviews(productId: string | undefined, sort?: 'newest' | 'highest' | 'lowest' | 'helpful') {
  return useQuery({ queryKey: ['reviews', productId, sort], queryFn: () => listReviewsForProduct(productId!, sort), enabled: !!productId })
}

export function useProductQuestions(productId: string | undefined) {
  return useQuery({ queryKey: ['questions', productId], queryFn: () => listQuestionsForProduct(productId!), enabled: !!productId })
}

export function useSubmitReview() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (review: Omit<Review, 'id' | 'helpfulCount' | 'notHelpfulCount' | 'createdAt'>) => submitReview(review),
    onSuccess: (review) => queryClient.invalidateQueries({ queryKey: ['reviews', review.productId] }),
  })
}

export function useMarkHelpful() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ reviewId, helpful }: { reviewId: string; helpful: boolean }) => markReviewHelpful(reviewId, helpful),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['reviews'] }),
  })
}

export function ratingDistribution(reviews: Review[]): RatingDistribution {
  const dist: RatingDistribution = { 5: 0, 4: 0, 3: 0, 2: 0, 1: 0 }
  for (const r of reviews) dist[r.rating] += 1
  return dist
}
