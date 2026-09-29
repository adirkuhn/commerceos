import { useTranslation } from 'react-i18next'
import { Package, Trash2, Edit3 } from 'lucide-react'
import clsx from 'clsx'
import { SKU_STATUS_STYLES, formatPrice } from './productFormat'
import { StockBadge } from './VariantsSection'
import type { Media, SKU, VariantWithSKUs } from './types'

interface VariantRowSummaryProps {
  variant: VariantWithSKUs
  sku: SKU | undefined
  selectedImage: Media | null
  // Without product media there's nothing to pick, so the thumbnail is
  // plain rather than a button.
  canPickImage: boolean
  onToggleImagePicker: () => void
  onEdit: () => void
  onRemove: () => void
}

// The always-visible line of a variant row: thumbnail, name/code, price,
// status, stock, and edit/remove actions.
export function VariantRowSummary({ variant, sku, selectedImage, canPickImage, onToggleImagePicker, onEdit, onRemove }: VariantRowSummaryProps) {
  const { t } = useTranslation('pms')
  const skuStatusLabel = { active: t('skuStatus.active'), inactive: t('skuStatus.inactive') }
  const thumbnailContent = selectedImage ? (
    <img src={selectedImage.url} alt={selectedImage.alt_text} className="w-full h-full object-cover" />
  ) : (
    <Package size={14} className="text-ink-4" />
  )

  return (
    <div className="flex items-center gap-3">
      {canPickImage ? (
        <button
          type="button"
          aria-label={t('variant.imageAria', { name: variant.name })}
          onClick={onToggleImagePicker}
          className="w-10 h-10 rounded-lg dark:bg-night-800 bg-gray-100 flex items-center justify-center shrink-0 overflow-hidden"
        >
          {thumbnailContent}
        </button>
      ) : (
        <div className="w-10 h-10 rounded-lg dark:bg-night-800 bg-gray-100 flex items-center justify-center shrink-0 overflow-hidden">
          {thumbnailContent}
        </div>
      )}
      <div className="flex-1 min-w-0">
        <p className="text-xs font-medium text-ink truncate">{variant.name}</p>
        <p className="text-2xs font-mono text-ink-4 truncate">
          {sku ? sku.sku_code : t('variant.noSkuYet')}
        </p>
      </div>
      {sku && sku.sale_price_minor_units != null ? (
        <span className="flex items-center gap-1.5 shrink-0">
          <span className="text-2xs text-ink-4 line-through">
            {formatPrice(sku.price_minor_units, sku.currency_code)}
          </span>
          <span className="text-xs font-medium text-success">
            {formatPrice(sku.sale_price_minor_units, sku.currency_code)}
          </span>
          {sku.sale_end_date && (
            <span className="text-2xs text-ink-4">{t('variant.until', { date: sku.sale_end_date })}</span>
          )}
        </span>
      ) : sku && (
        <span className="text-xs font-medium text-ink shrink-0">
          {formatPrice(sku.price_minor_units, sku.currency_code)}
        </span>
      )}
      {sku && (
        <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium capitalize shrink-0', SKU_STATUS_STYLES[sku.status])}>
          {skuStatusLabel[sku.status]}
        </span>
      )}
      {sku && <StockBadge skuId={sku.id} />}
      <div className="flex items-center gap-0.5 shrink-0">
        <button aria-label={t('variant.editVariant')}
          type="button" onClick={onEdit} title={t('variant.editVariant')}
          className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
        >
          <Edit3 size={12} className="text-ink-3" />
        </button>
        <button aria-label={t('variant.removeVariant')}
          type="button"
          onClick={onRemove}
          title={t('variant.removeVariant')}
          className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
        >
          <Trash2 size={12} className="text-ink-3 hover:text-danger" />
        </button>
      </div>
    </div>
  )
}
