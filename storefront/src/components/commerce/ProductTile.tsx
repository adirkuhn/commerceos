import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { ProductSummary } from '../../api/storefront'
import { useShopFormat } from '../../i18n/format'
import { useRenderEnv } from '../../runtime/RenderContext'
import { useCart } from '../../runtime/page/CartContext'

/** What a tile shows besides picture, name and link. */
export interface TileOptions {
  showPrice: boolean
  showDiscount?: boolean
  showStock?: boolean
  showAddToCart?: boolean
  /** The product's category above its name (needs category_name). */
  showCategory?: boolean
  /** full: wide button under the tile; compact: small "+ Add" by the price. */
  addButton?: 'full' | 'compact'
}

/** Whole-percent discount of a sale price, e.g. 38 for 500 → 310. */
export function percentOff(price: number | null | undefined, sale: number | null | undefined): number | undefined {
  if (price == null || sale == null || sale >= price || price <= 0) return undefined
  const pct = Math.round(((price - sale) / price) * 100)
  return pct > 0 ? pct : undefined
}

/** One product in a grid or card: picture, name, price, linking to its
 *  page; optionally a discount badge, stock status and add to cart. */
export function ProductTile({ product, showPrice, showDiscount = false, showStock = false, showAddToCart = false, showCategory = false, addButton = 'full' }: { product: ProductSummary } & TileOptions) {
  const env = useRenderEnv()
  const cart = useCart()
  const { t } = useTranslation()
  const { price } = useShopFormat()
  const [added, setAdded] = useState(false)
  const discount = percentOff(product.price_minor_units, product.sale_price_minor_units)
  const onSale = discount !== undefined
  const href = env.storePath(`products/${encodeURIComponent(product.slug)}`)
  const inert = env.mode === 'editor' ? { tabIndex: -1, onClick: (e: { preventDefault(): void }) => e.preventDefault() } : {}
  const known = product.in_stock !== undefined

  const add = async () => {
    if (!cart || !product.sku_code || env.mode === 'editor') return
    await cart.add(product.sku_code, 1)
    setAdded(true)
  }

  return (
    <div className={addButton === 'compact' ? 'cos-product-tile cos-tile-compact' : 'cos-product-tile'}>
      <a className="cos-product-link" href={href} {...inert}>
        <div className="cos-product-image">
          {product.image_url ? <img src={product.image_url} alt="" loading="lazy" /> : <span aria-hidden="true" />}
          {showDiscount && onSale && <span className="cos-badge-discount">{t('product.percentOff', { percent: discount })}</span>}
        </div>
        {showCategory && product.category_name && <span className="cos-product-category">{product.category_name}</span>}
        <span className="cos-product-name">{product.name}</span>
        {showPrice && product.price_minor_units != null && (
          <span className="cos-product-price">
            {onSale ? (
              <>
                <s aria-label={t('product.was')}>{price(product.price_minor_units, product.currency_code)}</s>{' '}
                <span className="cos-sale">{price(product.sale_price_minor_units, product.currency_code)}</span>
              </>
            ) : price(product.price_minor_units, product.currency_code)}
          </span>
        )}
      </a>
      {showStock && known && (
        <span className={product.in_stock ? 'cos-stock cos-in-stock' : 'cos-stock cos-out-of-stock'}>
          {product.in_stock ? t('product.inStock') : t('product.soldOut')}
        </span>
      )}
      {showAddToCart && known && (
        !product.in_stock ? (
          <button type="button" className="cos-button cos-button-outline cos-tile-action" disabled>{t('product.soldOut')}</button>
        ) : product.sku_code ? (
          <button type="button" className="cos-button cos-button-primary cos-tile-action" disabled={cart?.busy} onClick={() => void add()} {...inert}>
            {added && !cart?.busy ? t('product.added') : addButton === 'compact' ? <><span aria-hidden="true">+{'\u00a0'}</span>{t('product.add')}</> : t('product.addToCart')}
          </button>
        ) : (
          <a className="cos-button cos-button-outline cos-tile-action" href={href} {...inert}>{t('product.chooseOptions')}</a>
        )
      )}
    </div>
  )
}
