import { useSuspenseQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useShopFormat } from '../../i18n/format'
import { useStorefrontAPI } from '../../api/ApiContext'
import { useProductPage } from '../../runtime/page/ProductContext'
import { recommendationsQuery, reviewsQuery } from '../../runtime/page/queries'
import { EditorHint } from '../EditorHint'
import { reviewLimit } from '../data'
import type { RendererProps } from '../types'
import type { ProductRecommendationsProps, ProductReviewsProps } from './productPage'
import { ProductTile } from './ProductTile'

function Stars({ rating }: { rating: number }) {
  const full = Math.round(rating)
  const { t } = useTranslation()
  return <span className="cos-stars" role="img" aria-label={t('product.rating', { rating: rating.toFixed(1) })}>{'★'.repeat(full)}{'☆'.repeat(5 - full)}</span>
}

function Reviews({ productId, props, nodeId, className }: { productId: string; props: ProductReviewsProps; nodeId: string; className: string }) {
  const api = useStorefrontAPI()
  const { data } = useSuspenseQuery(reviewsQuery(api, productId, reviewLimit(props)))
  const { date } = useShopFormat()
  const { t } = useTranslation()
  return (
    <section className={`cos-reviews ${className}`} data-node={nodeId} aria-label={props.title || t('product.reviews')}>
      {props.title && <h2 className="cos-heading">{props.title}</h2>}
      {data.summary.count === 0 ? <p className="cos-muted">{t('product.noReviews')}</p> : (
        <>
          <p><Stars rating={data.summary.average} /> <span className="cos-muted">({data.summary.count})</span></p>
          <ul>
            {data.reviews.map(r => (
              <li key={r.id} className="cos-review">
                <Stars rating={r.rating} /> <strong>{r.title}</strong>
                <p>{r.body}</p>
                <p className="cos-muted">{r.customer_name}{r.verified ? ' · Verified buyer' : ''} · {date(r.created_at)}</p>
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  )
}

export function ProductReviewsView({ node, props, className }: RendererProps<ProductReviewsProps>) {
  const page = useProductPage()
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="reviews" />
  return <Reviews productId={page.product.id} props={props} nodeId={node.id} className={className} />
}

function Recommendations({ productId, props, nodeId, className }: { productId: string; props: ProductRecommendationsProps; nodeId: string; className: string }) {
  const api = useStorefrontAPI()
  const { data } = useSuspenseQuery(recommendationsQuery(api, productId))
  const { t } = useTranslation()
  const items = data.slice(0, Math.min(Math.max(Number(props.limit) || 4, 1), 12))
  if (!items.length) return null
  return (
    <section className={`cos-recommendations ${className}`} data-node={nodeId} aria-label={props.title || t('product.recommendations')}>
      {props.title && <h2 className="cos-heading">{props.title}</h2>}
      <div className="cos-product-grid">{items.map(p => <ProductTile key={p.id} product={p} showPrice />)}</div>
    </section>
  )
}

export function ProductRecommendationsView({ node, props, className }: RendererProps<ProductRecommendationsProps>) {
  const page = useProductPage()
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="recommendations" />
  return <Recommendations productId={page.product.id} props={props} nodeId={node.id} className={className} />
}
