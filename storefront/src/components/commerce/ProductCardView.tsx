import { useSuspenseQuery } from '@tanstack/react-query'
import { useStorefrontAPI } from '../../api/ApiContext'
import { useTranslation } from 'react-i18next'
import { useRenderEnv } from '../../runtime/RenderContext'
import type { RendererProps } from '../types'
import { productCardQuery } from '../data'
import type { ProductCardProps } from './productCard'
import { ProductTile } from './ProductTile'

export function ProductCardView({ node, props, className }: RendererProps<ProductCardProps>) {
  const api = useStorefrontAPI()
  const env = useRenderEnv()
  const { t } = useTranslation()
  const { data } = useSuspenseQuery(productCardQuery(api, props))
  if (!data) {
    return env.mode === 'editor'
      ? <div className={`cos-placeholder ${className}`} data-node={node.id}>{t('hint.chooseProduct')}</div>
      : null
  }
  return (
    <div className={`cos-product-card ${className}`} data-node={node.id}>
      <ProductTile product={data} showPrice={props.show_price !== false} />
    </div>
  )
}
