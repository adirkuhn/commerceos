import { useEffect, useState } from 'react'
import { Package } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import * as integrationsApi from '../integrations/api'
import type { Connection, BolProductOverride, BolCommission } from '../integrations/types'
import type { SKU } from './types'

// Pure so the arithmetic is unit-testable without rendering the component.
// All money inputs are major currency units (e.g. 19.99, not minor units) --
// the caller converts from SKU.price_minor_units/cost_price_minor_units once
// at the boundary.
export interface BolProfitInputs {
  sellPrice: number
  costPrice: number
  taxRatePercent: number
  commissionFixedAmount: number
  commissionPercent: number
  shippingCost: number
}

export interface BolProfitResult {
  taxAmount: number
  commissionAmount: number
  profit: number
  marginPercent: number
}

export function computeBolProfit(in_: BolProfitInputs): BolProfitResult {
  const taxAmount = in_.sellPrice * (in_.taxRatePercent / 100)
  const commissionAmount = in_.commissionFixedAmount + in_.sellPrice * (in_.commissionPercent / 100)
  const profit = in_.sellPrice - in_.costPrice - taxAmount - commissionAmount - in_.shippingCost
  const marginPercent = in_.sellPrice > 0 ? (profit / in_.sellPrice) * 100 : 0
  return { taxAmount, commissionAmount, profit, marginPercent }
}

function fmt(n: number): string {
  return n.toFixed(2)
}

const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900 outline-none focus:ring-1 focus:ring-blue-500'
const labelClass = 'block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5'

export function ProductBolTab({ productId, connection, sku }: { productId: string; connection: Connection; sku: SKU | null }) {
  const { token } = useAuth()

  const [, setOverride] = useState<BolProductOverride | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const [eanOverride, setEanOverride] = useState('')
  const [gpcCode, setGpcCode] = useState('')
  const [condition, setCondition] = useState<BolProductOverride['condition_category']>('NEW')
  const [referenceOverride, setReferenceOverride] = useState('')
  const [customPrice, setCustomPrice] = useState('')

  const [commission, setCommission] = useState<BolCommission | null>(null)
  const [commissionError, setCommissionError] = useState<string | null>(null)
  const [commissionOverridden, setCommissionOverridden] = useState(false)
  const [commissionPercentInput, setCommissionPercentInput] = useState('0')
  const [shippingCost, setShippingCost] = useState('0')
  const [taxRatePercent, setTaxRatePercent] = useState('0')

  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  // Bumped after a successful save so the commission effect below re-fetches
  // -- the live commission is price-bracket dependent, and a saved
  // CustomPriceMinorUnits changes the price the backend feeds Bol's API.
  const [commissionRefreshKey, setCommissionRefreshKey] = useState(0)

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setLoadError(null)
    integrationsApi.getBolProductOverride(token, connection.id, productId)
      .then(result => {
        if (cancelled) return
        setOverride(result)
        if (result) {
          setEanOverride(result.ean_override ?? '')
          setGpcCode(result.gpc_code ?? '')
          setCondition(result.condition_category)
          setReferenceOverride(result.reference_override ?? '')
          setCustomPrice(result.custom_price_minor_units != null ? fmt(result.custom_price_minor_units / 100) : '')
          if (result.commission_override_basis_points != null) {
            setCommissionOverridden(true)
            setCommissionPercentInput(fmt(result.commission_override_basis_points / 100))
          }
        }
      })
      .catch(err => { if (!cancelled) setLoadError(err instanceof Error ? err.message : 'Failed to load Bol overrides') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, connection.id, productId])

  useEffect(() => {
    if (!token || !sku) return
    let cancelled = false
    setCommissionError(null)
    integrationsApi.getBolCommission(token, connection.id, productId, sku.id)
      .then(result => {
        if (cancelled) return
        setCommission(result)
        if (!commissionOverridden) setCommissionPercentInput(fmt(result.percentage))
      })
      .catch(err => { if (!cancelled) setCommissionError(err instanceof Error ? err.message : 'Failed to fetch Bol commission') })
    return () => { cancelled = true }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, connection.id, productId, sku?.id, commissionRefreshKey])

  const handleSave = async () => {
    if (!token) return
    setSaveError(null)
    setSaving(true)
    try {
      const commissionPercent = parseFloat(commissionPercentInput)
      const parsedCustomPrice = parseFloat(customPrice)
      const result = await integrationsApi.upsertBolProductOverride(token, connection.id, productId, {
        eanOverride: eanOverride || null,
        gpcCode: gpcCode || null,
        conditionCategory: condition,
        fulfilmentMethod: 'FBR',
        commissionOverrideBasisPoints: commissionOverridden && !Number.isNaN(commissionPercent) ? Math.round(commissionPercent * 100) : null,
        referenceOverride: referenceOverride || null,
        customPriceMinorUnits: customPrice.trim() !== '' && !Number.isNaN(parsedCustomPrice) ? Math.round(parsedCustomPrice * 100) : null,
      })
      setOverride(result)
      setCommissionRefreshKey(k => k + 1)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  const defaultPrice = sku ? sku.price_minor_units / 100 : 0
  const parsedCustomPrice = parseFloat(customPrice)
  const sellPrice = customPrice.trim() !== '' && !Number.isNaN(parsedCustomPrice) ? parsedCustomPrice : defaultPrice
  const costPrice = sku?.cost_price_minor_units != null ? sku.cost_price_minor_units / 100 : 0
  const parsedCommissionPercent = parseFloat(commissionPercentInput) || 0
  const parsedShipping = parseFloat(shippingCost) || 0
  const parsedTaxRate = parseFloat(taxRatePercent) || 0
  const result = computeBolProfit({
    sellPrice, costPrice, taxRatePercent: parsedTaxRate,
    commissionFixedAmount: commission?.fixed_amount ?? 0,
    commissionPercent: parsedCommissionPercent,
    shippingCost: parsedShipping,
  })

  return (
    <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-4">
      <div className="flex items-center gap-2">
        <Package size={13} className="dark:text-gray-400 text-gray-500" />
        <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">Bol.com</h3>
      </div>

      {loading && <p className="text-[11px] dark:text-gray-500 text-gray-400">Loading…</p>}
      {!loading && loadError && <p className="text-[11px] dark:text-red-400 text-red-600">{loadError}</p>}

      {!loading && !loadError && (
        <>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label htmlFor="bol-ean-override" className={labelClass}>EAN override</label>
              <input id="bol-ean-override" value={eanOverride} onChange={e => setEanOverride(e.target.value)}
                placeholder={sku?.barcode ?? ''} className={inputClass} />
            </div>
            <div>
              <label htmlFor="bol-gpc-code" className={labelClass}>GPC Code</label>
              <input id="bol-gpc-code" value={gpcCode} onChange={e => setGpcCode(e.target.value)} className={inputClass} />
              <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-1">Only required if this EAN doesn't already exist in Bol's catalog.</p>
            </div>
            <div>
              <label htmlFor="bol-condition" className={labelClass}>Condition</label>
              <select id="bol-condition" value={condition} onChange={e => setCondition(e.target.value as BolProductOverride['condition_category'])} className={inputClass}>
                <option value="NEW">New</option>
                <option value="SECONDHAND">Secondhand</option>
                <option value="REFURBISHED">Refurbished</option>
              </select>
            </div>
            <div>
              <label className={labelClass}>Fulfilment method</label>
              <input disabled readOnly value="FBR (self-ship)" className={`${inputClass} cursor-not-allowed opacity-70`} />
            </div>
            <div>
              <label htmlFor="bol-reference" className={labelClass}>Reference (optional)</label>
              <input id="bol-reference" value={referenceOverride} onChange={e => setReferenceOverride(e.target.value)} className={inputClass} />
            </div>
            <div>
              <label htmlFor="bol-custom-price" className={labelClass}>Custom price (optional)</label>
              <input id="bol-custom-price" value={customPrice} onChange={e => setCustomPrice(e.target.value)}
                placeholder={fmt(defaultPrice)} className={inputClass} />
              <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-1">Leave empty to use the product's default price ({fmt(defaultPrice)}).</p>
            </div>
          </div>

          {saveError && <p className="text-[11px] text-red-400">{saveError}</p>}
          <button type="button" onClick={handleSave} disabled={saving}
            className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors">
            {saving ? 'Saving…' : 'Save'}
          </button>

          <div className="pt-3 border-t dark:border-[#2D3748] border-gray-200 space-y-3">
            <h4 className="text-xs font-semibold dark:text-gray-100 text-gray-900">Profit calculator</h4>
            {commissionError && <p className="text-[11px] dark:text-red-400 text-red-600">{commissionError}</p>}

            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className={labelClass}>Sell price</label>
                <input disabled readOnly value={fmt(sellPrice)} className={`${inputClass} cursor-not-allowed opacity-70`} />
                {customPrice.trim() !== '' && <p className="text-[10px] text-blue-400 mt-1">Custom price</p>}
              </div>
              <div>
                <label className={labelClass}>Cost price</label>
                <input disabled readOnly value={fmt(costPrice)} className={`${inputClass} cursor-not-allowed opacity-70`} />
              </div>
              <div>
                <label htmlFor="bol-tax-rate" className={labelClass}>Tax rate %</label>
                <input id="bol-tax-rate" value={taxRatePercent} onChange={e => setTaxRatePercent(e.target.value)} className={inputClass} />
              </div>
              <div>
                <label htmlFor="bol-commission-percent" className={labelClass}>
                  Bol commission % {commission && !commissionOverridden && '(live)'}
                </label>
                <input id="bol-commission-percent" value={commissionPercentInput}
                  onChange={e => { setCommissionPercentInput(e.target.value); setCommissionOverridden(true) }}
                  className={inputClass} />
                {commissionOverridden && (
                  <button type="button" className="text-[10px] text-blue-400 mt-1"
                    onClick={() => { setCommissionOverridden(false); if (commission) setCommissionPercentInput(fmt(commission.percentage)) }}>
                    Use live value
                  </button>
                )}
              </div>
              <div>
                <label className={labelClass}>Commission fixed fee</label>
                <input disabled readOnly value={fmt(commission?.fixed_amount ?? 0)} className={`${inputClass} cursor-not-allowed opacity-70`} />
              </div>
              <div>
                <label htmlFor="bol-shipping-cost" className={labelClass}>Shipping cost</label>
                <input id="bol-shipping-cost" value={shippingCost} onChange={e => setShippingCost(e.target.value)} className={inputClass} />
              </div>
            </div>

            <div className="rounded-lg dark:bg-[#111827] bg-gray-50 p-3 space-y-1 text-[11px]">
              <div className="flex justify-between dark:text-gray-400 text-gray-500"><span>Sell price</span><span>{fmt(sellPrice)}</span></div>
              <div className="flex justify-between dark:text-gray-400 text-gray-500"><span>− Cost price</span><span>{fmt(costPrice)}</span></div>
              <div className="flex justify-between dark:text-gray-400 text-gray-500"><span>− Tax</span><span>{fmt(result.taxAmount)}</span></div>
              <div className="flex justify-between dark:text-gray-400 text-gray-500"><span>− Bol commission</span><span>{fmt(result.commissionAmount)}</span></div>
              <div className="flex justify-between dark:text-gray-400 text-gray-500"><span>− Shipping</span><span>{fmt(parsedShipping)}</span></div>
              <div className="flex justify-between font-semibold dark:text-gray-100 text-gray-900 pt-1 border-t dark:border-[#2D3748] border-gray-200">
                <span>Profit</span><span>{fmt(result.profit)}</span>
              </div>
              <div className="flex justify-between dark:text-gray-400 text-gray-500">
                <span>Margin</span><span>{fmt(result.marginPercent)}%</span>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
