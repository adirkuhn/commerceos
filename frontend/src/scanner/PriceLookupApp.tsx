import { useState } from 'react'
import { ArrowLeft, Tag, Boxes } from 'lucide-react'
import * as pmsApi from '../pms/api'
import * as inventoryApi from '../inventory/api'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { ScanCodeScreen } from './ScanCodeScreen'
import { ProductCandidateRow } from './ProductCandidateRow'
import type { SKU } from '../pms/types'
import type { Resolved } from './InventoryApp'

interface PriceLookupAppProps {
  token: string
  onExit: () => void
  // Lets "View in Inventory" jump straight to that mini-app's SKU screen
  // for the already-resolved SKU, instead of the caller having to re-scan.
  onViewInventory: (resolved: Resolved) => void
}

function formatPrice(minorUnits: number, currency: string): string {
  return `${currency} ${(minorUnits / 100).toFixed(2)}`
}

interface Result {
  sku: SKU
  productName: string
  stockTotal: number | null
}

// Scanner mini-app: scan/type a code, see price + stock, done -- read-only,
// no confirm step needed. Also works by tapping alone via the on-screen
// keyboard into the same input a hardware scan-engine keyboard-wedges into.
export function PriceLookupApp({ token, onExit, onViewInventory }: PriceLookupAppProps) {
  const [result, setResult] = useState<Result | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [candidates, setCandidates] = useState<SKUCandidate[]>([])

  async function showResult(resolvedSKU: SKU, productName: string) {
    let stockTotal: number | null = null
    try {
      stockTotal = (await inventoryApi.getSKUStockLevel(token, resolvedSKU.id)).total
    } catch {
      // Stock lookup failing shouldn't hide the price -- just omit it.
    }
    setResult({ sku: resolvedSKU, productName, stockTotal })
  }

  // A camera misread or a hand-typed code can be truncated/mistyped -- on an
  // exact-match miss, fall back to a partial (substring) match on product
  // name, SKU code, or barcode (see searchSKUCandidates) instead of just
  // failing.
  async function handleSubmit(code: string) {
    setError('')
    setCandidates([])
    setLoading(true)
    try {
      const { sku, product_name } = await pmsApi.resolveBarcode(token, code)
      await showResult(sku, product_name)
    } catch (err) {
      try {
        const found = await searchSKUCandidates(token, code)
        if (found.length === 1) {
          await showResult(found[0].sku, found[0].productName)
          return
        }
        if (found.length > 1) {
          setCandidates(found)
          return
        }
      } catch {
        // fall through to the original resolveBarcode error below
      }
      setError(err instanceof Error ? err.message : 'Failed to resolve code')
    } finally {
      setLoading(false)
    }
  }

  function reset() {
    setResult(null)
    setCandidates([])
    setError('')
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
        <button onClick={onExit} className="p-1 -ml-1" title="Back to apps">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Price Lookup</h1>
      </div>

      {!result && (
        <div className="flex-1 flex flex-col overflow-y-auto">
          <ScanCodeScreen icon={Tag} prompt="Scan a barcode or type a SKU code" loading={loading} error={error} onSubmit={handleSubmit} />
          {candidates.length > 0 && (
            <div className="w-full max-w-xs mx-auto px-6 pb-6 space-y-1.5 shrink-0">
              <p className="text-xs text-gray-500">Multiple matches -- pick one:</p>
              {candidates.map(c => (
                <ProductCandidateRow key={c.sku.id} token={token} candidate={c} onClick={() => showResult(c.sku, c.productName)} />
              ))}
            </div>
          )}
        </div>
      )}

      {result && (
        <div className="flex-1 flex flex-col items-center justify-center gap-6 p-6 text-center">
          <div className="w-16 h-16 rounded-2xl bg-blue-600/20 flex items-center justify-center">
            <Tag size={28} className="text-blue-400" />
          </div>
          <div>
            <p className="text-lg font-semibold">{result.productName}</p>
            <p className="text-xs text-gray-500">{result.sku.sku_code}</p>
          </div>
          <div className="space-y-1">
            {result.sku.sale_price_minor_units != null ? (
              <>
                <p className="text-3xl font-bold text-blue-400">{formatPrice(result.sku.sale_price_minor_units, result.sku.currency_code)}</p>
                <p className="text-sm text-gray-500 line-through">{formatPrice(result.sku.price_minor_units, result.sku.currency_code)}</p>
              </>
            ) : (
              <p className="text-3xl font-bold text-gray-100">{formatPrice(result.sku.price_minor_units, result.sku.currency_code)}</p>
            )}
          </div>
          {result.stockTotal !== null && (
            <p className="text-sm text-gray-400">In stock: <span className="font-mono text-gray-200">{result.stockTotal}</span></p>
          )}
          <div className="w-full max-w-xs space-y-2">
            <button
              onClick={() => onViewInventory({ sku: result.sku, productName: result.productName })}
              className="w-full flex items-center justify-center gap-1.5 py-3 rounded-xl bg-blue-600 text-white font-medium"
            >
              <Boxes size={16} /> View in Inventory
            </button>
            <button
              onClick={reset}
              className="w-full py-3 rounded-xl bg-[#1F2937] text-gray-200 font-medium border border-[#2D3748]"
            >
              Look up another
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
