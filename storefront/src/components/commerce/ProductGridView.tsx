import { useSuspenseQuery } from '@tanstack/react-query'
import { useStorefrontAPI } from '../../api/ApiContext'
import { useTranslation } from 'react-i18next'
import { useRenderEnv } from '../../runtime/RenderContext'
import type { RendererProps } from '../types'
import { productGridQuery } from '../data'
import type { ProductGridProps } from './productGrid'
import { ProductTile } from './ProductTile'

export function ProductGridView({ node, props, className }: RendererProps<ProductGridProps>) {
  const api = useStorefrontAPI()
  const env = useRenderEnv()
  const { t } = useTranslation()
  const { data } = useSuspenseQuery(productGridQuery(api, props))
  const cols = Number(props.columns) || 0
  if (!data.items.length) {
    return env.mode === 'editor'
      ? <div className={`cos-placeholder ${className}`} data-node={node.id}>{t('hint.noProductsYet')}</div>
      : null
  }
  return (
    <div className={`cos-product-grid ${className}`} data-node={node.id}
      style={cols ? ({ '--cos-grid-cols': cols } as React.CSSProperties) : undefined}>
      {data.items.map(p => <ProductTile key={p.id} product={p} showPrice={props.show_price !== false} showDiscount={props.show_discount !== false} showStock={props.show_stock === true} showAddToCart={props.show_add_to_cart === true} />)}
    </div>
  )
}
