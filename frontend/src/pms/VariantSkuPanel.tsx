import { useState } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { BarcodeAliasesEditor, ScaleCodeAliasesEditor } from './SkuAliasEditors'
import { MarginInsightCard } from './ProductPricingCards'
import { VariantSkuWeightPoolControls } from './VariantSkuWeightPoolControls'
import type { SkuBarcodeEdit } from './hooks/useSkuBarcodeEdit'
import type { SKU, UpdateSKUPayload } from './types'

interface VariantSkuPanelProps {
  token: string
  sku: SKU
  barcode: SkuBarcodeEdit
  updateSKU: (id: string, payload: UpdateSKUPayload) => Promise<void>
  onPortionsCreated: () => void
  onRemove: () => void
  taxRateBasisPoints: number | null
  pricesIncludeTax: boolean
}

// Inline controls under a variant that already has a SKU: its EAN,
// alternate codes, weight-pool link, removal, and cost/margin insight.
export function VariantSkuPanel({ token, sku, barcode, updateSKU, onPortionsCreated, onRemove, taxRateBasisPoints, pricesIncludeTax }: VariantSkuPanelProps) {
  const [showMargin, setShowMargin] = useState(false)
  return (
    <div className="mt-1.5 space-y-1">
      <div className="flex items-center gap-1.5">
        <label htmlFor={`variant-sku-barcode-${sku.id}`} className="text-2xs text-ink-3 shrink-0">
          EAN
        </label>
        <input
          id={`variant-sku-barcode-${sku.id}`} type="text"
          placeholder={sku.scale_code ? 'Not used — see scale code below' : 'Barcode / EAN'}
          disabled={!!sku.scale_code} value={sku.scale_code ? '' : barcode.value} onChange={e => barcode.setValue(e.target.value)}
          className="flex-1 min-w-0 px-2 py-1 text-2xs rounded bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
        />
        {!sku.scale_code && barcode.value !== (sku.barcode ?? '') && (
          <button
            type="button" disabled={barcode.saving} onClick={barcode.save}
            className="text-2xs text-accent hover:text-accent/80 transition-colors disabled:opacity-50 shrink-0"
          >
            {barcode.saving ? 'Saving…' : 'Save'}
          </button>
        )}
      </div>
      {barcode.error && <p className="text-2xs text-danger">{barcode.error}</p>}
      {sku.scale_code ? <ScaleCodeAliasesEditor skuId={sku.id} /> : <BarcodeAliasesEditor skuId={sku.id} />}
      <VariantSkuWeightPoolControls
        token={token} sku={sku} updateSKU={updateSKU} onPortionsCreated={onPortionsCreated}
        trailing={
          <button
            type="button"
            onClick={onRemove}
            className="text-2xs text-ink-4 hover:text-danger transition-colors"
          >
            Remove SKU
          </button>
        }
      />
      <button
        type="button" onClick={() => setShowMargin(v => !v)}
        className="flex items-center gap-1 text-2xs font-medium text-ink-3 hover:text-ink-2 transition-colors"
      >
        {showMargin ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
        Cost &amp; margin insight
      </button>
      {showMargin && (
        <MarginInsightCard
          skuId={sku.id} currency={sku.currency_code}
          taxRateBasisPoints={taxRateBasisPoints}
          pricesIncludeTax={pricesIncludeTax}
        />
      )}
    </div>
  )
}
