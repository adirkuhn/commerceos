import { useTranslation } from 'react-i18next'
import { useRenderEnv } from '../../runtime/RenderContext'
import { useListingPage } from '../../runtime/page/ListingContext'
import { useProductPage } from '../../runtime/page/ProductContext'
import { EditorHint } from '../EditorHint'
import type { RendererProps } from '../types'
import type { BreadcrumbsProps, CollectionFiltersProps, CollectionHeaderProps, CollectionProductsProps } from './listing'
import { ProductTile } from './ProductTile'

export function CollectionHeaderView({ node, props, className }: RendererProps<CollectionHeaderProps>) {
  const page = useListingPage()
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="listing" />
  return (
    <header className={`cos-collection-header ${className}`} data-node={node.id}>
      {props.show_image !== false && page.imageUrl && <img src={page.imageUrl} alt="" className="cos-collection-image" />}
      <h1 className="cos-heading">{page.title}</h1>
      {props.show_description !== false && page.description && <p className="cos-muted">{page.description}</p>}
    </header>
  )
}

export function CollectionProductsView({ node, props, className }: RendererProps<CollectionProductsProps>) {
  const page = useListingPage()
  const { t } = useTranslation()
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="listing" />
  const cols = Number(props.columns) || 0
  if (!page.listing.items.length) {
    return <p className={`cos-muted ${className}`} data-node={node.id}>{t('listing.noProducts')}</p>
  }
  return (
    <div className={`cos-product-grid ${className}`} data-node={node.id} style={cols ? ({ '--cos-grid-cols': cols } as React.CSSProperties) : undefined}>
      {page.listing.items.map(p => <ProductTile key={p.id} product={p} showPrice={props.show_price !== false} />)}
    </div>
  )
}

export function CollectionFiltersView({ node, props, className }: RendererProps<CollectionFiltersProps>) {
  const page = useListingPage()
  const { t } = useTranslation()
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="listing" />
  const facets = page.listing.category_facets ?? []
  if (!facets.length) return null
  return (
    <nav className={`cos-filters ${className}`} data-node={node.id} aria-label={props.title || t('listing.filter')}>
      {props.title && <h2 className="cos-label">{props.title}</h2>}
      <ul>
        <li><a href={page.hrefForCategory(null)} aria-current={!page.activeCategoryId ? 'true' : undefined}>{t('listing.all')}</a></li>
        {facets.map(f => (
          <li key={f.category_id}>
            <a href={page.hrefForCategory(f.category_id)} aria-current={page.activeCategoryId === f.category_id ? 'true' : undefined}>
              {f.category_name} <span className="cos-muted">({f.count})</span>
            </a>
          </li>
        ))}
      </ul>
    </nav>
  )
}

export function PaginationView({ node, className }: RendererProps) {
  const page = useListingPage()
  const { t } = useTranslation()
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="pagination" />
  if (page.pageCount <= 1) return null
  return (
    <nav className={`cos-pagination ${className}`} data-node={node.id} aria-label={t('listing.pages')}>
      {page.page > 1 && <a href={page.hrefForPage(page.page - 1)} rel="prev">{t('listing.previous')}</a>}
      <span>{t('listing.pageOf', { page: page.page, total: page.pageCount })}</span>
      {page.page < page.pageCount && <a href={page.hrefForPage(page.page + 1)} rel="next">{t('listing.next')}</a>}
    </nav>
  )
}

export function BreadcrumbsView({ node, props, className }: RendererProps<BreadcrumbsProps>) {
  const env = useRenderEnv()
  const product = useProductPage()
  const listing = useListingPage()
  const { t } = useTranslation()
  const current = product?.product.name ?? listing?.title
  if (!current) return <EditorHint nodeId={node.id} className={className} hint="breadcrumbs" />
  return (
    <nav className={`cos-breadcrumbs ${className}`} data-node={node.id} aria-label={t('nav.breadcrumb')}>
      <ol>
        <li><a href={env.storePath('')}>{props.home_label || t('nav.home')}</a></li>
        <li aria-current="page">{current}</li>
      </ol>
    </nav>
  )
}
