import { useState } from 'react'
import { useForm, Controller } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Star, ThumbsUp, ThumbsDown, BadgeCheck } from 'lucide-react'
import { reviewSchema, type ReviewFormValues } from '@/lib/validation'
import { useProductReviews, useSubmitReview, useMarkHelpful, ratingDistribution } from '@/features/reviews/hooks'
import { useAuthStore } from '@/stores/authStore'
import { formatDate, cn } from '@/lib/utils'
import { Button } from '@/components/ui/Button'
import { Textarea } from '@/components/ui/Textarea'
import { Input } from '@/components/ui/Input'
import { Label } from '@/components/ui/Label'
import { Select } from '@/components/ui/Select'
import { LoadingState } from '@/components/ui/States'
import { toast } from '@/components/ui/toast'

export function ReviewsSection({ productId }: { productId: string }) {
  const [sort, setSort] = useState<'newest' | 'highest' | 'lowest' | 'helpful'>('newest')
  const { data: reviews = [], isLoading } = useProductReviews(productId, sort)
  const markHelpful = useMarkHelpful()
  const [formOpen, setFormOpen] = useState(false)
  const customer = useAuthStore(s => s.session?.customer)

  const distribution = ratingDistribution(reviews)
  const total = reviews.length
  const average = total > 0 ? reviews.reduce((sum, r) => sum + r.rating, 0) / total : 0

  return (
    <div className="grid gap-8 lg:grid-cols-[16rem_1fr]">
      <div>
        <p className="text-4xl font-bold">{average.toFixed(1)}</p>
        <div className="my-1 flex" aria-hidden="true">
          {Array.from({ length: 5 }, (_, i) => <Star key={i} className={i < Math.round(average) ? 'h-4 w-4 fill-amber-400 text-amber-400' : 'h-4 w-4 text-muted-foreground'} />)}
        </div>
        <p className="mb-4 text-sm text-muted-foreground">Based on {total} reviews</p>
        <div className="space-y-1.5">
          {([5, 4, 3, 2, 1] as const).map(star => (
            <div key={star} className="flex items-center gap-2 text-xs">
              <span className="w-8">{star} star</span>
              <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-secondary">
                <div className="h-full bg-amber-400" style={{ width: total > 0 ? `${(distribution[star] / total) * 100}%` : '0%' }} />
              </div>
              <span className="w-6 text-right text-muted-foreground">{distribution[star]}</span>
            </div>
          ))}
        </div>
        <Button className="mt-4 w-full" variant="outline" onClick={() => setFormOpen(o => !o)}>Write a review</Button>
      </div>

      <div>
        <div className="mb-4 flex items-center justify-between">
          <p className="text-sm font-medium">{total} reviews</p>
          <Select aria-label="Sort reviews" value={sort} onChange={(e) => setSort(e.target.value as typeof sort)} className="w-40">
            <option value="newest">Newest</option>
            <option value="highest">Highest rated</option>
            <option value="lowest">Lowest rated</option>
            <option value="helpful">Most helpful</option>
          </Select>
        </div>

        {formOpen && (
          <ReviewForm
            productId={productId}
            author={customer ? `${customer.firstName} ${customer.lastName[0]}.` : ''}
            onDone={() => setFormOpen(false)}
          />
        )}

        {isLoading ? <LoadingState /> : reviews.length === 0 ? (
          <p className="py-8 text-sm text-muted-foreground">No reviews yet. Be the first to share your thoughts.</p>
        ) : (
          <ul className="divide-y divide-border">
            {reviews.map(review => (
              <li key={review.id} className="py-5">
                <div className="mb-1 flex items-center gap-2">
                  <div className="flex" aria-hidden="true">
                    {Array.from({ length: 5 }, (_, i) => <Star key={i} className={cn('h-3.5 w-3.5', i < review.rating ? 'fill-amber-400 text-amber-400' : 'text-muted-foreground')} />)}
                  </div>
                  {review.verifiedPurchase && (
                    <span className="flex items-center gap-1 text-xs text-emerald-600"><BadgeCheck className="h-3.5 w-3.5" /> Verified purchase</span>
                  )}
                </div>
                <p className="text-sm font-semibold">{review.title}</p>
                <p className="mt-1 text-sm text-muted-foreground">{review.body}</p>
                <p className="mt-2 text-xs text-muted-foreground">{review.author} · {formatDate(review.createdAt)}</p>
                <div className="mt-2 flex items-center gap-3 text-xs text-muted-foreground">
                  <span>Helpful?</span>
                  <button className="flex items-center gap-1 hover:text-foreground" onClick={() => markHelpful.mutate({ reviewId: review.id, helpful: true })}>
                    <ThumbsUp className="h-3.5 w-3.5" /> {review.helpfulCount}
                  </button>
                  <button className="flex items-center gap-1 hover:text-foreground" onClick={() => markHelpful.mutate({ reviewId: review.id, helpful: false })}>
                    <ThumbsDown className="h-3.5 w-3.5" /> {review.notHelpfulCount}
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

function ReviewForm({ productId, author, onDone }: { productId: string; author: string; onDone: () => void }) {
  const submitReview = useSubmitReview()
  const [authorName, setAuthorName] = useState(author)
  const { register, handleSubmit, control, formState: { errors } } = useForm<ReviewFormValues>({
    resolver: zodResolver(reviewSchema),
    defaultValues: { rating: 5, title: '', body: '' },
  })

  return (
    <form
      className="mb-6 space-y-3 rounded-[var(--radius)] border border-border p-4"
      noValidate
      onSubmit={handleSubmit((values) => {
        submitReview.mutate(
          { productId, author: authorName || 'Anonymous', rating: values.rating as 1 | 2 | 3 | 4 | 5, title: values.title, body: values.body, verifiedPurchase: false },
          { onSuccess: () => { toast({ title: 'Review submitted', variant: 'success' }); onDone() } },
        )
      })}
    >
      <div>
        <Label htmlFor="review-name">Your name</Label>
        <Input id="review-name" value={authorName} onChange={(e) => setAuthorName(e.target.value)} className="mt-1.5" />
      </div>
      <div>
        <Label>Rating</Label>
        <Controller
          control={control} name="rating"
          render={({ field }) => (
            <div className="mt-1.5 flex gap-1" role="radiogroup" aria-label="Rating">
              {[1, 2, 3, 4, 5].map(n => (
                <button key={n} type="button" aria-label={`${n} star`} aria-pressed={field.value === n} onClick={() => field.onChange(n)}>
                  <Star className={cn('h-6 w-6', n <= field.value ? 'fill-amber-400 text-amber-400' : 'text-muted-foreground')} />
                </button>
              ))}
            </div>
          )}
        />
      </div>
      <div>
        <Label htmlFor="review-title">Title</Label>
        <Input id="review-title" {...register('title')} className="mt-1.5" />
        {errors.title && <p className="mt-1 text-xs text-destructive">{errors.title.message}</p>}
      </div>
      <div>
        <Label htmlFor="review-body">Review</Label>
        <Textarea id="review-body" {...register('body')} className="mt-1.5" rows={4} />
        {errors.body && <p className="mt-1 text-xs text-destructive">{errors.body.message}</p>}
      </div>
      <Button type="submit" disabled={submitReview.isPending}>{submitReview.isPending ? 'Submitting…' : 'Submit review'}</Button>
    </form>
  )
}
