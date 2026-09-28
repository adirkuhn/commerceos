import type { ProductSummary } from '../../api/storefront'
import { formatPrice } from '../../core/money'
import { useRenderEnv } from '../../runtime/RenderContext'

/** One product in a grid or card: picture, name, price, linking to its page. */
export function ProductTile({ product, showPrice }: { product: ProductSummary; showPrice: boolean }) {
  const env = useRenderEnv()
  const onSale = product.sale_price_minor_units != null && product.sale_price_minor_units < (product.price_minor_units ?? Infinity)
  const editorLink = env.mode === 'editor' ? { tabIndex: -1, onClick: (e: { preventDefault(): void }) => e.preventDefault() } : {}
  return (
    <a className="cos-product-tile" href={env.storePath(`products/${encodeURIComponent(product.slug)}`)} {...editorLink}>
      <div className="cos-product-image">
        {product.image_url ? <img src={product.image_url} alt="" loading="lazy" /> : <span aria-hidden="true" />}
      </div>
      <span className="cos-product-name">{product.name}</span>
      {showPrice && product.price_minor_units != null && (
        <span className="cos-product-price">
          {onSale ? (
            <>
              <s aria-label="Was">{formatPrice(product.price_minor_units, product.currency_code)}</s>{' '}
              <span className="cos-sale">{formatPrice(product.sale_price_minor_units, product.currency_code)}</span>
            </>
          ) : formatPrice(product.price_minor_units, product.currency_code)}
        </span>
      )}
    </a>
  )
}
