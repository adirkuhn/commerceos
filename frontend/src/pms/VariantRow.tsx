import { useState } from 'react'
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
      setImageError(err instanceof Error ? err.message : 'Failed to update image')
    }
  }

  const removeVariant = async () => {
    if (await confirm({ title: `Remove variant "${variant.name}"?`, message: 'Its SKU and stock records are removed too.', confirmLabel: 'Remove variant', destructive: true })) {
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
            if (await confirm({ title: `Remove SKU "${sku.sku_code}"?`, message: 'Its stock records are removed too.', confirmLabel: 'Remove SKU', destructive: true })) {
              detail.removeSKU(sku.id)
            }
          }}
          taxRateBasisPoints={taxRateForSku?.rate_basis_points ?? null}
          pricesIncludeTax={tenantSettings?.prices_include_tax ?? false}
        />
      )}

      {!sku && !addSku.open && (
        <button type="button" onClick={() => addSku.setOpen(true)} className="text-2xs text-accent hover:text-accent/80 transition-colors mt-2">
          + Add SKU
        </button>
      )}

      {addSku.open && <VariantAddSkuPanel addSku={addSku} />}
      {confirmDialog}
    </div>
  )
}
