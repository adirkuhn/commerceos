import { useState } from 'react'
import type { SKU, UpdateSKUPayload } from '../types'
import { skuPassThroughPayload } from '../skuPayloads'

// Inline edit of an existing SKU's barcode (Bol, for one, treats it as the
// EAN, so it must stay fixable after creation). Saves immediately, sending
// every other field back unchanged. Prefilled once, the first time the row
// has a SKU.
export function useSkuBarcodeEdit(sku: SKU | undefined, updateSKU: (id: string, payload: UpdateSKUPayload) => Promise<void>) {
  const [value, setValue] = useState('')
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  if (!hydrated && sku) {
    setValue(sku.barcode ?? '')
    setHydrated(true)
  }

  const save = async () => {
    if (!sku) return
    setError('')
    setSaving(true)
    try {
      await updateSKU(sku.id, { ...skuPassThroughPayload(sku), barcode: value || undefined })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update barcode')
    } finally {
      setSaving(false)
    }
  }

  return { value, setValue, saving, error, save }
}

export type SkuBarcodeEdit = ReturnType<typeof useSkuBarcodeEdit>
