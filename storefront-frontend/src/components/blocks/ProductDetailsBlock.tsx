import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { getProduct } from '../../api/products'
import * as reviewsApi from '../../api/reviews'
import { useCart } from '../../cart/CartContext'
import { useWishlist } from '../../wishlist/WishlistContext'
import { formatMoney } from '../../format'
import type { StorefrontProduct, Review, ReviewSummary, SubmitReviewInput } from '../../types'

export function ProductDetailsBlock() {
  const { id } = useParams<{ id: string }>()
  const { addItem } = useCart()
  const { isSaved, toggle } = useWishlist()
  const [product, setProduct] = useState<StorefrontProduct | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [addedSku, setAddedSku] = useState<string | null>(null)

  useEffect(() => {
    if (!id) {
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    getProduct(id)
      .then(p => { if (!cancelled) setProduct(p) })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Unknown error') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [id])

  if (!id) return <p className="p-4 text-sm text-red-600">No product selected.</p>
  if (loading) return <p className="p-4 text-sm text-gray-500">Loading product…</p>
  if (error) return <p className="p-4 text-sm text-red-600">Could not load this product: {error}</p>
  if (!product) return null

  const handleAdd = async (skuCode: string) => {
    setAddedSku(null)
    await addItem(skuCode, 1)
    setAddedSku(skuCode)
  }

  return (
    <div className="h-full overflow-auto p-6">
      <div className="flex items-start justify-between gap-3">
        <h1 className="text-2xl font-semibold text-brand-700">{product.name}</h1>
        <button
          onClick={() => toggle(product.id)}
          aria-label={isSaved(product.id) ? 'Remove from wishlist' : 'Save to wishlist'}
          aria-pressed={isSaved(product.id)}
          className="shrink-0 rounded-lg border border-gray-200 px-3 py-1.5 text-xs font-medium text-brand-500 transition-colors hover:bg-gray-50"
        >
          {isSaved(product.id) ? '♥ Saved' : '♡ Save'}
        </button>
      </div>
      {product.description && <p className="mt-2 text-brand-500">{product.description}</p>}
      <div className="mt-6 space-y-4">
        {product.variants.map(variant => (
          <div key={variant.id}>
            <p className="text-sm font-medium text-brand-700">{variant.name}</p>
            <div className="mt-2 space-y-2">
              {variant.skus.map(sku => (
                <div key={sku.id} className="flex items-center justify-between rounded-lg border border-gray-200 p-3">
                  <div>
                    <p className="text-sm text-brand-700">{sku.sku_code}</p>
                    <p className="text-sm text-brand-500">
                      {sku.sale_price_minor_units != null ? (
                        <>
                          <span className="mr-1 text-gray-400 line-through">{formatMoney(sku.price_minor_units, sku.currency_code)}</span>
                          {formatMoney(sku.sale_price_minor_units, sku.currency_code)}
                        </>
                      ) : formatMoney(sku.price_minor_units, sku.currency_code)}
                    </p>
                  </div>
                  <button
                    onClick={() => handleAdd(sku.sku_code)}
                    disabled={sku.stock_quantity <= 0}
                    className="rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-brand-700 disabled:opacity-50"
                  >
                    {sku.stock_quantity <= 0 ? 'Out of stock' : addedSku === sku.sku_code ? 'Added' : 'Add to cart'}
                  </button>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
      <ReviewsSection productId={product.id} />
    </div>
  )
}

function Stars({ rating }: { rating: number }) {
  return (
    <span aria-label={`${rating} out of 5 stars`}>
      {'★'.repeat(rating)}
      <span className="text-gray-300">{'★'.repeat(5 - rating)}</span>
    </span>
  )
}

function ReviewsSection({ productId }: { productId: string }) {
  const [reviews, setReviews] = useState<Review[]>([])
  const [summary, setSummary] = useState<ReviewSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showForm, setShowForm] = useState(false)

  const load = () => {
    setLoading(true)
    setError(null)
    reviewsApi.listReviews(productId)
      .then(result => { setReviews(result.reviews); setSummary(result.summary) })
      .catch(e => setError(e instanceof Error ? e.message : 'Unknown error'))
      .finally(() => setLoading(false))
  }

  useEffect(load, [productId])

  return (
    <div className="mt-10 border-t border-gray-100 pt-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-brand-700">Reviews</h2>
          {summary && summary.count > 0 && (
            <p className="mt-1 text-sm text-brand-500">
              <Stars rating={Math.round(summary.average)} /> {summary.average.toFixed(1)} out of 5 ({summary.count} review{summary.count === 1 ? '' : 's'})
            </p>
          )}
        </div>
        {!showForm && (
          <button
            onClick={() => setShowForm(true)}
            className="rounded-lg border border-brand-600 px-3 py-1.5 text-xs font-medium text-brand-700 transition-colors hover:bg-brand-50"
          >
            Write a review
          </button>
        )}
      </div>

      {showForm && (
        <ReviewForm
          productId={productId}
          onSubmitted={() => { setShowForm(false); load() }}
          onCancel={() => setShowForm(false)}
        />
      )}

      {loading && <p className="mt-4 text-sm text-gray-500">Loading reviews…</p>}
      {!loading && error && <p className="mt-4 text-sm text-red-600">{error}</p>}
      {!loading && !error && reviews.length === 0 && (
        <p className="mt-4 text-sm text-brand-500">No reviews yet -- be the first to write one.</p>
      )}
      <div className="mt-4 space-y-4">
        {reviews.map(r => (
          <div key={r.id} className="border-b border-gray-100 pb-4">
            <div className="flex items-center gap-2">
              <Stars rating={r.rating} />
              <p className="text-sm font-medium text-brand-700">{r.customer_name}</p>
              {r.verified && <span className="rounded-full bg-green-50 px-2 py-0.5 text-[10px] font-medium text-green-700">Verified purchase</span>}
            </div>
            {r.title && <p className="mt-1 text-sm font-medium text-brand-700">{r.title}</p>}
            <p className="mt-1 text-sm text-brand-500">{r.body}</p>
          </div>
        ))}
      </div>
    </div>
  )
}

function ReviewForm({ productId, onSubmitted, onCancel }: { productId: string; onSubmitted: () => void; onCancel: () => void }) {
  const [customerName, setCustomerName] = useState('')
  const [customerEmail, setCustomerEmail] = useState('')
  const [rating, setRating] = useState(5)
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    try {
      const input: SubmitReviewInput = { customer_name: customerName, customer_email: customerEmail, rating, title, body }
      await reviewsApi.submitReview(productId, input)
      onSubmitted()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setSaving(false)
    }
  }

  const inputClass = 'w-full rounded border border-gray-200 px-2 py-1.5 text-sm'

  return (
    <form onSubmit={handleSubmit} className="mt-4 space-y-3 rounded-lg border border-gray-200 p-4">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor="review-name" className="mb-1 block text-xs font-medium text-brand-500">Name</label>
          <input id="review-name" required value={customerName} onChange={e => setCustomerName(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label htmlFor="review-email" className="mb-1 block text-xs font-medium text-brand-500">Email</label>
          <input id="review-email" type="email" required value={customerEmail} onChange={e => setCustomerEmail(e.target.value)} className={inputClass} />
        </div>
      </div>
      <div>
        <label htmlFor="review-rating" className="mb-1 block text-xs font-medium text-brand-500">Rating</label>
        <select id="review-rating" value={rating} onChange={e => setRating(Number(e.target.value))} className={inputClass}>
          {[5, 4, 3, 2, 1].map(n => <option key={n} value={n}>{n} star{n === 1 ? '' : 's'}</option>)}
        </select>
      </div>
      <div>
        <label htmlFor="review-title" className="mb-1 block text-xs font-medium text-brand-500">Title (optional)</label>
        <input id="review-title" value={title} onChange={e => setTitle(e.target.value)} className={inputClass} />
      </div>
      <div>
        <label htmlFor="review-body" className="mb-1 block text-xs font-medium text-brand-500">Review</label>
        <textarea id="review-body" required rows={3} value={body} onChange={e => setBody(e.target.value)} className={inputClass} />
      </div>
      {error && <p className="text-xs text-red-600">{error}</p>}
      <div className="flex items-center gap-2">
        <button type="submit" disabled={saving} className="rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-brand-700 disabled:opacity-50">
          {saving ? 'Submitting…' : 'Submit review'}
        </button>
        <button type="button" onClick={onCancel} className="rounded-lg border border-gray-200 px-3 py-1.5 text-xs font-medium text-brand-500 hover:bg-gray-50">
          Cancel
        </button>
      </div>
    </form>
  )
}
