import { useState, useEffect } from 'react'
import { Search, X, Boxes } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { useApp } from '../context/AppContext'
import { useProducts } from './hooks/useProducts'
import { useProductDetail } from './hooks/useProductDetail'
import { useProductMedia } from './hooks/useProductMedia'
import * as pmsApi from './api'
import * as inventoryApi from '../inventory/api'
import type { SKU } from './types'
import type { StockLevel } from '../inventory/types'
import { VariantRow } from './VariantRow'
import { linkedSkuLabel } from './linkedSkuLabel'

// ── Variants + SKUs section (only shown when the product has variants) ───────

interface VariantsSectionProps {
  detail: ReturnType<typeof useProductDetail>
  productId: string
}

export function VariantsSection({ detail, productId }: VariantsSectionProps) {
  const { token } = useAuth()
  const { media: productMedia } = useProductMedia(token ?? '', productId)
  const [adding, setAdding] = useState(false)
  const [variantName, setVariantName] = useState('')
  const [addError, setAddError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const resetForm = () => {
    setAdding(false)
    setVariantName('')
    setAddError('')
  }

  const handleAddVariant = async () => {
    setAddError('')
    setSubmitting(true)
    try {
      await detail.addVariant({ name: variantName })
    } catch (err) {
      setAddError(err instanceof Error ? err.message : 'Failed to add variant')
      setSubmitting(false)
      return
    }
    setSubmitting(false)
    resetForm()
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-end">
        <button
          type="button"
          onClick={() => setAdding(v => !v)}
          className="text-2xs font-medium px-2 py-1 rounded-lg dark:bg-night-800 bg-gray-100 text-ink-2 hover:bg-hover transition-colors"
        >
          {adding ? 'Cancel' : '+ Add Variant'}
        </button>
      </div>

      {detail.loading && <p className="text-xs text-ink-4">Loading…</p>}
      {detail.error && <p className="text-xs text-danger">{detail.error}</p>}

      <div className="space-y-2">
        {detail.detail?.variants.map(v => (
          <VariantRow key={v.id} variant={v} detail={detail} productMedia={productMedia} />
        ))}
      </div>

      {adding && (
        // A plain div, not a <form> — this already lives inside the page's
        // outer <form id="product-general-form">, and nested <form>
        // elements are invalid HTML. Browsers silently break out of the
        // inner one, so its submit button ends up submitting the OUTER
        // form as a real (non-JS-intercepted) navigation — a full page
        // reload back to the app's initial route, which looked like the
        // "Add variant" button redirecting home.
        <div className="p-3 rounded-lg border border-line space-y-2">
          <input
            type="text" required placeholder="Variant name (e.g. Large)" value={variantName} onChange={e => setVariantName(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); handleAddVariant() } }}
            className="w-full px-2.5 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
          />
          {addError && <p className="text-xs text-danger">{addError}</p>}
          <button type="button" onClick={handleAddVariant} disabled={submitting} className="w-full py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors">
            {submitting ? 'Adding…' : 'Add variant'}
          </button>
          <p className="text-2xs text-ink-4">Add a SKU (price/barcode) to the variant after creating it.</p>
        </div>
      )}
    </div>
  )
}

interface CaseOfSKUPickerProps {
  // The SKU being edited can't be a case of itself -- filtered out of the
  // results client-side (the backend also rejects it, see
  // ErrCaseOfSKUCannotBeSelf, but hiding it up front avoids a round trip).
  excludeSKUID: string
  // Extra client-side filter beyond excludeSKUID -- e.g. only kilo
  // (unit_type=kilo) SKUs when picking a weight pool to portion from.
  // Hides invalid choices up front instead of letting the backend reject
  // them after the fact.
  filterSku?: (sku: SKU) => boolean
  onSelect: (sku: SKU, label: string) => void
  onClose: () => void
}

export function CaseOfSKUPicker({ excludeSKUID, filterSku, onSelect, onClose }: CaseOfSKUPickerProps) {
  const { token } = useAuth()
  const [query, setQuery] = useState('')
  const { products } = useProducts(token ?? '', { search: query, perPage: 5 })
  const [expandedProductId, setExpandedProductId] = useState<string | null>(null)
  const [skus, setSkus] = useState<SKU[]>([])
  const [loadingSkus, setLoadingSkus] = useState(false)

  async function toggleProduct(productId: string) {
    if (expandedProductId === productId) {
      setExpandedProductId(null)
      return
    }
    setExpandedProductId(productId)
    setLoadingSkus(true)
    try {
      const detail = await pmsApi.getProduct(token ?? '', productId)
      const candidates = detail.variants.flatMap(v => v.skus ?? []).filter(s => s.id !== excludeSKUID)
      setSkus(filterSku ? candidates.filter(filterSku) : candidates)
    } finally {
      setLoadingSkus(false)
    }
  }

  return (
    <div className="rounded-lg border border-line dark:bg-night-950 bg-white p-3 space-y-2">
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            autoFocus value={query} onChange={e => setQuery(e.target.value)}
            placeholder="Search products by name…"
            className="w-full pl-7 pr-2 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <button aria-label="Close" type="button" onClick={onClose} title="Close">
          <X size={14} className="text-ink-4" />
        </button>
      </div>
      <div className="max-h-48 overflow-auto space-y-1">
        {products.length === 0 && <p className="text-2xs text-ink-4 px-1">No matching products</p>}
        {products.map(p => (
          <div key={p.id}>
            <button
              type="button"
              onClick={() => toggleProduct(p.id)}
              className="w-full text-left px-2 py-1.5 text-xs rounded hover:bg-hover text-ink"
            >
              {p.name}
            </button>
            {expandedProductId === p.id && (
              <div className="pl-4 space-y-0.5">
                {loadingSkus && <p className="text-2xs text-ink-4">Loading…</p>}
                {!loadingSkus && skus.length === 0 && <p className="text-2xs text-ink-4">No other SKUs on this product</p>}
                {!loadingSkus && skus.map(s => (
                  <button
                    key={s.id}
                    type="button"
                    onClick={() => onSelect(s, linkedSkuLabel({ product_name: p.name, sku_code: s.sku_code, barcode: s.barcode }))}
                    className="w-full flex items-center justify-between text-left px-2 py-1 text-2xs rounded hover:bg-hover"
                  >
                    <span className="font-mono text-ink-2">{s.sku_code}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

// Shows a SKU's current stock level (fetched live, not cached with the
// product) and a shortcut into the Inventory page for batches/adjustments
// — kept read-only here since Inventory is the single place stock actions
// happen, to avoid duplicating that UI.
export function StockBadge({ skuId }: { skuId: string }) {
  const { token } = useAuth()
  const { setActiveNav } = useApp()
  const [level, setLevel] = useState<StockLevel | null>(null)

  useEffect(() => {
    if (!token) return
    let cancelled = false
    inventoryApi.getSKUStockLevel(token, skuId).then(l => { if (!cancelled) setLevel(l) }).catch(() => {})
    return () => { cancelled = true }
  }, [token, skuId])

  return (
    <button
      type="button"
      onClick={() => setActiveNav('inventory')}
      title="View in Inventory"
      className="flex items-center gap-1 text-2xs text-ink-3 hover:text-accent shrink-0"
    >
      <Boxes size={11} />
      {level ? `${level.total} in stock` : '…'}
    </button>
  )
}
