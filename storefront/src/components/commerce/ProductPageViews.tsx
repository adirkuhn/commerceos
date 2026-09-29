import { createElement, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useShopFormat } from '../../i18n/format'
import { htmlToParagraphs } from '../../core/plainText'
import { useRenderEnv } from '../../runtime/RenderContext'
import { useCart } from '../../runtime/page/CartContext'
import { useProductPage } from '../../runtime/page/ProductContext'
import { EditorHint } from '../EditorHint'
import type { RendererProps } from '../types'
import type {
  AddToCartProps, ProductGalleryProps, ProductPriceProps, ProductTitleProps, QuantitySelectorProps, VariantSelectorProps,
} from './productPage'

export function ProductGalleryView({ node, props, className }: RendererProps<ProductGalleryProps>) {
  const page = useProductPage()
  const { t } = useTranslation()
  const [active, setActive] = useState(0)
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="product" />
  const images = page.product.images
  if (!images.length) return <div className={`cos-gallery cos-gallery-empty ${className}`} data-node={node.id} aria-hidden="true" />
  if (props.layout === 'stacked') {
    return (
      <div className={`cos-gallery cos-gallery-stacked ${className}`} data-node={node.id}>
        {images.map((src, i) => <img key={src} src={src} alt={i === 0 ? page.product.name : ''} loading={i === 0 ? 'eager' : 'lazy'} />)}
      </div>
    )
  }
  const current = images[Math.min(active, images.length - 1)]
  return (
    <div className={`cos-gallery ${className}`} data-node={node.id}>
      <img className="cos-gallery-main" src={current} alt={page.product.name} />
      {images.length > 1 && (
        <div className="cos-gallery-thumbs">
          {images.map((src, i) => (
            <button key={src} type="button" aria-label={t('product.showImage', { number: i + 1 })} aria-pressed={i === active} onClick={() => setActive(i)}>
              <img src={src} alt="" loading="lazy" />
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

export function ProductTitleView({ node, props, className }: RendererProps<ProductTitleProps>) {
  const page = useProductPage()
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="product" />
  const level = [1, 2, 3].includes(Number(props.level)) ? Number(props.level) : 1
  return createElement(`h${level}`, { className: `cos-heading cos-product-title ${className}`, 'data-node': node.id }, page.product.name)
}

export function ProductPriceView({ node, props, className }: RendererProps<ProductPriceProps>) {
  const page = useProductPage()
  const { t } = useTranslation()
  const { price } = useShopFormat()
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="product" />
  const sku = page.sku
  if (!sku) return null
  const onSale = sku.sale_price_minor_units != null && sku.sale_price_minor_units < sku.price_minor_units
  return (
    <p className={`cos-price ${className}`} data-node={node.id}>
      {onSale ? (
        <>
          <span className="cos-sale">{price(sku.sale_price_minor_units, sku.currency_code)}</span>
          {props.show_compare_at !== false && <> <s aria-label={t('product.was')}>{price(sku.price_minor_units, sku.currency_code)}</s></>}
        </>
      ) : price(sku.price_minor_units, sku.currency_code)}
    </p>
  )
}

export function VariantSelectorView({ node, props, className }: RendererProps<VariantSelectorProps>) {
  const page = useProductPage()
  const { t } = useTranslation()
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="product" />
  const variants = page.product.variants
  if (variants.length < 2) return null
  const soldOut = (id: string) => !variants.find(v => v.id === id)?.skus.some(s => s.stock_quantity > 0)
  if (props.style === 'dropdown') {
    return (
      <label className={`cos-variants ${className}`} data-node={node.id}>
        <span className="cos-label">{t('product.option')}</span>
        <select value={page.variant?.id} onChange={e => page.selectVariant(e.target.value)}>
          {variants.map(v => <option key={v.id} value={v.id}>{v.name}{soldOut(v.id) ? ' (sold out)' : ''}</option>)}
        </select>
      </label>
    )
  }
  return (
    <fieldset className={`cos-variants ${className}`} data-node={node.id}>
      <legend className="cos-label">{t('product.option')}</legend>
      <div className="cos-variant-buttons">
        {variants.map(v => (
          <button key={v.id} type="button" aria-pressed={page.variant?.id === v.id} className={soldOut(v.id) ? 'cos-sold-out' : ''}
            onClick={() => page.selectVariant(v.id)}>{v.name}</button>
        ))}
      </div>
    </fieldset>
  )
}

export function QuantitySelectorView({ node, props, className }: RendererProps<QuantitySelectorProps>) {
  const page = useProductPage()
  const { t } = useTranslation()
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="product" />
  const max = Math.min(Math.max(Number(props.max) || 10, 1), 99)
  const set = (n: number) => page.setQuantity(Math.min(max, Math.max(1, n)))
  return (
    <div className={`cos-quantity ${className}`} data-node={node.id} role="group" aria-label={t('product.quantity')}>
      <button type="button" aria-label={t('product.fewer')} onClick={() => set(page.quantity - 1)} disabled={page.quantity <= 1}>−</button>
      <input type="number" inputMode="numeric" aria-label={t('product.quantity')} min={1} max={max} value={page.quantity} onChange={e => set(Number(e.target.value))} />
      <button type="button" aria-label={t('product.more')} onClick={() => set(page.quantity + 1)} disabled={page.quantity >= max}>+</button>
    </div>
  )
}

export function AddToCartView({ node, props, className }: RendererProps<AddToCartProps>) {
  const page = useProductPage()
  const { t } = useTranslation()
  const cart = useCart()
  const env = useRenderEnv()
  const [added, setAdded] = useState(false)
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="product" />
  const sku = page.sku
  const available = !!sku && sku.stock_quantity > 0
  const label = available ? props.label || t('product.addToCart') : props.sold_out_label || t('product.soldOut')
  const onClick = async () => {
    if (!sku || !cart || env.mode === 'editor') return
    setAdded(false)
    await cart.add(sku.sku_code, page.quantity)
    setAdded(true)
  }
  return (
    <div className={`cos-add-to-cart ${className}`} data-node={node.id}>
      <button type="button" className="cos-button cos-button-primary" disabled={!available || cart?.busy} onClick={() => void onClick()}>
        {label}
      </button>
      <p role="status" aria-live="polite" className="cos-status">
        {cart?.error ? <span className="cos-error">{cart.error}</span> : added && !cart?.busy ? t('product.added') : ''}
      </p>
    </div>
  )
}

export function ProductDescriptionView({ node, className }: RendererProps) {
  const page = useProductPage()
  if (!page) return <EditorHint nodeId={node.id} className={className} hint="product" />
  const paragraphs = htmlToParagraphs(page.product.description)
  if (!paragraphs.length) return null
  return <div className={`cos-text cos-description ${className}`} data-node={node.id}>{paragraphs.map((p, i) => <p key={i}>{p}</p>)}</div>
}
