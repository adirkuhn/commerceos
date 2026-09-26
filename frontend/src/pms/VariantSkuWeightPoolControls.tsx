import { useState, type ReactNode } from 'react'
import { BulkAddWeightPortionsModal } from './BulkAddWeightPortionsModal'
import { CaseOfSKUPicker } from './VariantsSection'
import { skuPassThroughPayload } from './skuPayloads'
import type { SKU, SKUUnitType, UpdateSKUPayload } from './types'
import { linkedSkuLabel } from './linkedSkuLabel'

interface VariantSkuWeightPoolControlsProps {
  token: string
  sku: SKU
  updateSKU: (id: string, payload: UpdateSKUPayload) => Promise<void>
  onPortionsCreated: () => void
  // Rendered at the right end of the controls line (the row's Remove SKU).
  trailing: ReactNode
}

// An existing variant SKU's weight-pool link: make/turn off a pool, link
// to/unlink from one, and bulk-add weighed portions from a scale-priced
// pool. Edits save immediately -- there's no batched "Save" step for an
// already-created variant's SKU, unlike the flat single-SKU form on the
// product's General tab.
export function VariantSkuWeightPoolControls({ token, sku, updateSKU, onPortionsCreated, trailing }: VariantSkuWeightPoolControlsProps) {
  const [bulkSaving, setBulkSaving] = useState(false)
  const [bulkError, setBulkError] = useState('')
  const [existingPortionPickerOpen, setExistingPortionPickerOpen] = useState(false)
  const [bulkAddOpen, setBulkAddOpen] = useState(false)

  const saveBulkPool = async (next: { unitType: SKUUnitType; portionedFromSkuId: string | null }) => {
    setBulkError('')
    setBulkSaving(true)
    try {
      await updateSKU(sku.id, { ...skuPassThroughPayload(sku), unit_type: next.unitType, portioned_from_sku_id: next.portionedFromSkuId })
      setExistingPortionPickerOpen(false)
    } catch (err) {
      setBulkError(err instanceof Error ? err.message : 'Failed to update weight pool link')
    } finally {
      setBulkSaving(false)
    }
  }

  return (
    <>
      <div className="flex items-center justify-between">
        {sku.unit_type === 'kilo' ? (
          <span className="text-2xs text-ink-3 flex items-center gap-1.5">
            Weight pool
            <button type="button" disabled={bulkSaving} onClick={() => saveBulkPool({ unitType: 'unit', portionedFromSkuId: null })} className="hover:underline disabled:opacity-50">
              Turn off
            </button>
            {sku.pricing_mode === 'per_weight' && sku.scale_code && sku.price_per_kilo_minor_units != null && (
              <button type="button" onClick={() => setBulkAddOpen(true)} className="text-accent hover:text-accent/80 transition-colors">
                Bulk add by weight
              </button>
            )}
          </span>
        ) : sku.portioned_from_sku_id ? (
          <span className="text-2xs text-ink-3 flex items-center gap-1.5">
            Portioned from <span className="font-medium">{sku.portioned_from_sku ? linkedSkuLabel(sku.portioned_from_sku) : sku.portioned_from_sku_id}</span>
            <button type="button" disabled={bulkSaving} onClick={() => saveBulkPool({ unitType: sku.unit_type ?? 'unit', portionedFromSkuId: null })} className="hover:underline disabled:opacity-50">
              Unlink
            </button>
          </span>
        ) : existingPortionPickerOpen ? null : (
          <div className="flex items-center gap-2">
            <button type="button" disabled={bulkSaving} onClick={() => saveBulkPool({ unitType: 'kilo', portionedFromSkuId: null })} className="text-2xs text-accent hover:text-accent/80 transition-colors disabled:opacity-50">
              Make weight pool
            </button>
            <button type="button" onClick={() => setExistingPortionPickerOpen(true)} className="text-2xs text-accent hover:text-accent/80 transition-colors">
              Link to weight pool
            </button>
          </div>
        )}
        {trailing}
      </div>
      {existingPortionPickerOpen && (
        <CaseOfSKUPicker
          excludeSKUID={sku.id}
          filterSku={s => s.unit_type === 'kilo'}
          onSelect={(s) => saveBulkPool({ unitType: sku.unit_type ?? 'unit', portionedFromSkuId: s.id })}
          onClose={() => setExistingPortionPickerOpen(false)}
        />
      )}
      {bulkAddOpen && sku.price_per_kilo_minor_units != null && (
        <BulkAddWeightPortionsModal
          token={token} bulkSkuId={sku.id}
          pricePerKiloMinorUnits={sku.price_per_kilo_minor_units} currencyCode={sku.currency_code}
          onCreated={onPortionsCreated} onClose={() => setBulkAddOpen(false)}
        />
      )}
      {bulkError && <p className="text-2xs text-danger">{bulkError}</p>}
    </>
  )
}
