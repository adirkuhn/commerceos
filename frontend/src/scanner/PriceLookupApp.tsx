import { useEffect, useState, type ReactNode } from 'react'
import { Route, Routes } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { ArrowLeft, Tag, Boxes } from 'lucide-react'
import * as pmsApi from '../pms/api'
import * as inventoryApi from '../inventory/api'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { ScanCodeScreen } from './ScanCodeScreen'
import { ProductCandidateRow } from './ProductCandidateRow'
import { SkuRoute } from './SkuRoute'
import { scannerPaths } from './scannerRoutes'
import { useStackNavigation } from '../routing/useStackNavigation'
import type { Resolved } from './InventoryScanScreens'
import { apiErrorText } from '../api/apiErrorText'

interface PriceLookupAppProps {
  token: string
  onExit: () => void
}

function formatPrice(minorUnits: number, currency: string): string {
  return `${currency} ${(minorUnits / 100).toFixed(2)}`
}

// Scanner mini-app: scan/type a code, see price + stock, done -- read-only,
// no confirm step needed. Also works by tapping alone via the on-screen
// keyboard into the same input a hardware scan-engine keyboard-wedges into.
//
// Mounted at /scanner/price-lookup/* -- the scan screen is the index route,
// a result is /scanner/price-lookup/:code (see ./scannerRoutes).
export function PriceLookupApp({ token, onExit }: PriceLookupAppProps) {
  const { open } = useStackNavigation()
  const show = (r: Resolved) => open(scannerPaths.priceLookupSku(r.sku.sku_code))

  return (
    <Routes>
      <Route index element={<PriceLookupShell onExit={onExit}><PriceLookupScan token={token} onResolved={show} /></PriceLookupShell>} />
      <Route
        path=":code"
        element={
          <SkuRoute token={token} backTo={scannerPaths.priceLookup}>
            {(resolved, { back }) => (
              <PriceLookupShell onExit={onExit}>
                <PriceLookupResult
                  token={token}
                  resolved={resolved}
                  // "View in Inventory" jumps straight to that mini-app's SKU
                  // screen for this SKU instead of making the operator re-scan.
                  onViewInventory={() => open(scannerPaths.inventorySku(resolved.sku.sku_code))}
                  onLookUpAnother={back}
                />
              </PriceLookupShell>
            )}
          </SkuRoute>
        }
      />
    </Routes>
  )
}

function PriceLookupShell({ onExit, children }: { onExit: () => void; children: ReactNode }) {
  const { t } = useTranslation('scanner')
  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label={t('shared.backToApps')} onClick={onExit} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title={t('shared.backToApps')}>
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">{t('price.title')}</h1>
      </div>
      {children}
    </div>
  )
}

function PriceLookupScan({ token, onResolved }: { token: string; onResolved: (r: Resolved) => void }) {
  const { t } = useTranslation('scanner')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [candidates, setCandidates] = useState<SKUCandidate[]>([])

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
      onResolved({ sku, productName: product_name })
    } catch (err) {
      try {
        const found = await searchSKUCandidates(token, code)
        if (found.length === 1) {
          onResolved({ sku: found[0].sku, productName: found[0].productName })
          return
        }
        if (found.length > 1) {
          setCandidates(found)
          return
        }
      } catch {
        // fall through to the original resolveBarcode error below
      }
      setError(apiErrorText(err) ?? t('errors.resolveCode'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex-1 flex flex-col overflow-y-auto">
      <ScanCodeScreen icon={Tag} prompt={t('shared.scanPrompt')} loading={loading} error={error} onSubmit={handleSubmit} />
      {candidates.length > 0 && (
        <div className="w-full max-w-xs mx-auto px-6 pb-6 space-y-1.5 shrink-0">
          <p className="text-xs text-gray-400">{t('shared.multipleMatches')}</p>
          {candidates.map(c => (
            <ProductCandidateRow key={c.sku.id} token={token} candidate={c} onClick={() => onResolved({ sku: c.sku, productName: c.productName })} />
          ))}
        </div>
      )}
    </div>
  )
}

interface PriceLookupResultProps {
  token: string
  resolved: Resolved
  onViewInventory: () => void
  onLookUpAnother: () => void
}

function PriceLookupResult({ token, resolved, onViewInventory, onLookUpAnother }: PriceLookupResultProps) {
  const { t } = useTranslation('scanner')
  const { sku, productName } = resolved
  const [stockTotal, setStockTotal] = useState<number | null>(null)

  useEffect(() => {
    let current = true
    inventoryApi.getSKUStockLevel(token, sku.id)
      .then(level => { if (current) setStockTotal(level.total) })
      // Stock lookup failing shouldn't hide the price -- just omit it.
      .catch(() => { if (current) setStockTotal(null) })
    return () => { current = false }
  }, [token, sku.id])

  return (
    <div className="flex-1 flex flex-col items-center justify-center gap-6 p-6 text-center">
      <div className="w-16 h-16 rounded-2xl bg-accent/20 flex items-center justify-center">
        <Tag size={28} className="text-accent" />
      </div>
      <div>
        <p className="text-lg font-semibold">{productName}</p>
        <p className="text-xs text-gray-400">{sku.sku_code}</p>
      </div>
      <div className="space-y-1">
        {sku.sale_price_minor_units != null ? (
          <>
            <p className="text-3xl font-bold text-accent">{formatPrice(sku.sale_price_minor_units, sku.currency_code)}</p>
            <p className="text-sm text-gray-400 line-through">{formatPrice(sku.price_minor_units, sku.currency_code)}</p>
          </>
        ) : (
          <p className="text-3xl font-bold text-gray-100">{formatPrice(sku.price_minor_units, sku.currency_code)}</p>
        )}
      </div>
      {stockTotal !== null && (
        <p className="text-sm text-gray-400"><Trans t={t} i18nKey="price.inStock" values={{ quantity: stockTotal }} components={{ value: <span className="font-mono text-gray-200" /> }} /></p>
      )}
      <div className="w-full max-w-xs space-y-2">
        <button
          onClick={onViewInventory}
          className="w-full flex items-center justify-center gap-1.5 py-3 rounded-xl bg-accent-solid text-white font-medium"
        >
          <Boxes size={16} /> {t('price.viewInventory')}
        </button>
        <button
          onClick={onLookUpAnother}
          className="w-full py-3 rounded-xl bg-night-800 text-gray-200 font-medium border border-night-700"
        >
          {t('price.lookUpAnother')}
        </button>
      </div>
    </div>
  )
}
