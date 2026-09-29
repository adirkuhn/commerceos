import i18next from 'i18next'
import { useState } from 'react'
import type { SKU, UpdateSKUPayload } from '../types'
import { skuPassThroughPayload } from '../skuPayloads'

// Inline edit of an existing SKU's barcode (Bol, for one, treats it as the
// EAN, so it must stay fixable after creation). Saves immediately, sending
// every other field back unchanged. Prefilled per SKU: when the row gets a
// different SKU (one removed, another added without the row remounting),
// the field and any error reset to the new SKU's; a refresh of the same SKU
// (e.g. after saving) leaves what was typed alone.
export function useSkuBarcodeEdit(sku: SKU | undefined, updateSKU: (id: string, payload: UpdateSKUPayload) => Promise<void>) {
  const [value, setValue] = useState('')
  const [prefilledFor, setPrefilledFor] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  if (sku && sku.id !== prefilledFor) {
    setValue(sku.barcode ?? '')
    setError('')
    setPrefilledFor(sku.id)
  }

  const save = async () => {
    if (!sku) return
    setError('')
    setSaving(true)
    try {
      await updateSKU(sku.id, { ...skuPassThroughPayload(sku), barcode: value || undefined })
    } catch (err) {
      setError(err instanceof Error ? err.message : i18next.t('pms:errors.updateBarcode'))
    } finally {
      setSaving(false)
    }
  }

  return { value, setValue, saving, error, save }
}

export type SkuBarcodeEdit = ReturnType<typeof useSkuBarcodeEdit>
