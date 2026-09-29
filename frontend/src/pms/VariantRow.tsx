import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useProductDetail } from './hooks/useProductDetail'
import type { VariantWithSKUs, Media } from './types'
import { useConfirm } from '../components/common/useConfirm'
import { useSkuTaxContext } from './hooks/useSkuTaxContext'
import { useVariantAddSku } from './hooks/useVariantAddSku'
import { useSkuBarcodeEdit } from './hooks/useSkuBarcodeEdit'
import { VariantRowSummary } from './VariantRowSummary'
import { VariantEditPanel } from './VariantEditPanel'
import { VariantImagePicker } from './VariantImagePicker'
import { VariantSkuPanel } from './VariantSkuPanel'
import { VariantAddSkuPanel } from './VariantAddSkuPanel'

interface VariantRowProps {
  variant: VariantWithSKUs
  detail: ReturnType<typeof useProductDetail>
  productMedia: Media[]
}

// One variant in the variants manager: its summary line plus the inline
// panels (edit, image picker, existing-SKU controls, add SKU). Every write
// saves immediately through `detail`, which refreshes the product.
export function VariantRow({ variant, detail, productMedia }: VariantRowProps) {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { t } = useTranslation('pms')
  // Same "resolve tax class -> country rate" lookup as the product editor's
  // price preview, but scoped to this row's own SKU.tax_class_id.
  const { tenantSettings, rateFor } = useSkuTaxContext()
  const sku = variant.skus?.[0]
  const taxRateForSku = rateFor(sku?.tax_class_id)

  const addSku = useVariantAddSku(variant.id, detail.addSKU, tenantSettings)
  const barcode = useSkuBarcodeEdit(sku, detail.updateSKU)

  // Bumped on every "Edit" click so the panel remounts, re-prefilled from
  // the current variant/SKU.
  const [editKey, setEditKey] = useState(0)
  const [editing, setEditing] = useState(false)
  const openEdit = () => {
    setEditKey(k => k + 1)
    setEditing(true)
  }

  const [pickerOpen, setPickerOpen] = useState(false)
  const [imageError, setImageError] = useState('')
  const selectedImage = productMedia.find(m => m.id === variant.featured_media_id) ?? null

  const handleSelectImage = async (mediaId: string | null) => {
    setImageError('')
    try {
      await detail.updateVariant(variant.id, {
        name: variant.name,
        position: variant.position,
        featured_media_id: mediaId,
      })
      setPickerOpen(false)
    } catch (err) {
      setImageError(err instanceof Error ? err.message : t('variant.imageFailed'))
    }
  }

  const removeVariant = async () => {
    if (await confirm({ title: t('variant.removeTitle', { name: variant.name }), message: t('variant.removeMessage'), confirmLabel: t('variant.removeVariant'), destructive: true })) {
      detail.removeVariant(variant.id)
    }
  }

  return (
    <div className="p-3 rounded-lg border border-line">
      <VariantRowSummary
        variant={variant} sku={sku} selectedImage={selectedImage} canPickImage={productMedia.length > 0}
        onToggleImagePicker={() => setPickerOpen(v => !v)} onEdit={openEdit} onRemove={removeVariant}
      />

      {editing && (
        <VariantEditPanel
          key={editKey} variant={variant} sku={sku}
          updateVariant={detail.updateVariant} updateSKU={detail.updateSKU} onClose={() => setEditing(false)}
        />
      )}

      {pickerOpen && (
        <VariantImagePicker media={productMedia} selectedId={variant.featured_media_id} onSelect={handleSelectImage} />
      )}

      {imageError && <p className="text-2xs text-danger mt-1.5">{imageError}</p>}

      {sku && (
        <VariantSkuPanel
          token={token ?? ''} sku={sku} barcode={barcode} updateSKU={detail.updateSKU}
          onPortionsCreated={() => detail.refresh()}
          onRemove={async () => {
            if (await confirm({ title: t('variant.removeSkuTitle', { code: sku.sku_code }), message: t('variant.removeSkuMessage'), confirmLabel: t('variant.removeSku'), destructive: true })) {
              detail.removeSKU(sku.id)
            }
          }}
          taxRateBasisPoints={taxRateForSku?.rate_basis_points ?? null}
          pricesIncludeTax={tenantSettings?.prices_include_tax ?? false}
        />
      )}

      {!sku && !addSku.open && (
        <button type="button" onClick={() => addSku.setOpen(true)} className="text-2xs text-accent hover:text-accent/80 transition-colors mt-2">
          {t('variant.addSku')}
        </button>
      )}

      {addSku.open && <VariantAddSkuPanel addSku={addSku} />}
      {confirmDialog}
    </div>
  )
}
