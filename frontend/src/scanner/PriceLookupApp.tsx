import { useState } from 'react'
import { ArrowLeft, Tag } from 'lucide-react'
import * as pmsApi from '../pms/api'
import * as inventoryApi from '../inventory/api'
import { ScanCodeScreen } from './ScanCodeScreen'
import type { SKU } from '../pms/types'

interface PriceLookupAppProps {
  token: string
  onExit: () => void
}

function formatPrice(minorUnits: number, currency: string): string {
  return `${currency} ${(minorUnits / 100).toFixed(2)}`
}

interface Result {
  sku: SKU
  stockTotal: number | null
}

// Scanner mini-app: scan/type a code, see price + stock, done -- read-only,
// no confirm step needed. Also works by tapping alone via the on-screen
// keyboard into the same input a hardware scan-engine keyboard-wedges into.
export function PriceLookupApp({ token, onExit }: PriceLookupAppProps) {
  const [result, setResult] = useState<Result | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(code: string) {
    setError('')
    setLoading(true)
    try {
      const { sku } = await pmsApi.resolveBarcode(token, code)
      let stockTotal: number | null = null
      try {
        stockTotal = (await inventoryApi.getSKUStockLevel(token, sku.id)).total
      } catch {
        // Stock lookup failing shouldn't hide the price -- just omit it.
      }
      setResult({ sku, stockTotal })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to resolve code')
    } finally {
      setLoading(false)
    }
  }

  function reset() {
    setResult(null)
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
        <ScanCodeScreen icon={Tag} prompt="Scan a barcode or type a SKU code" loading={loading} error={error} onSubmit={handleSubmit} />
      )}

      {result && (
        <div className="flex-1 flex flex-col items-center justify-center gap-6 p-6 text-center">
          <div className="w-16 h-16 rounded-2xl bg-blue-600/20 flex items-center justify-center">
            <Tag size={28} className="text-blue-400" />
          </div>
          <p className="text-lg font-semibold">{result.sku.sku_code}</p>
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
          <button
            onClick={reset}
            className="w-full max-w-xs py-3 rounded-xl bg-[#1F2937] text-gray-200 font-medium border border-[#2D3748]"
          >
            Look up another
          </button>
        </div>
      )}
    </div>
  )
}
