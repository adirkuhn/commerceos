import { useEffect, useState } from 'react'
import { Package } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import * as integrationsApi from '../integrations/api'
import type { Connection, BolProductOverride, BolCommission } from '../integrations/types'
import type { UpsertBolProductContentPayload } from '../integrations/api'
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

function defaultBolProductContent(): UpsertBolProductContentPayload {
  return {
    manufacturer_contact: '', manufacturer_address: '', manufacturer_name: '',
    shelf_life_closed: '', shelf_life_after_opening: '', ingredients: '',
    content_amount: '', content_unit: '', country_of_origin: '', substance: '',
    instruction_language: '', packaging_language: '',
    items_per_package: null, package_insert_url: '', is_gift: false, dermatologically_tested: false,
    hazard_statements: '', skin_type: '', hypoallergenic: false, contains_zinc_oxide: false,
    content_per_dispenser_amount: '', content_per_dispenser_unit: '', brand: '',
    fragrance_free: false, ph_neutral: false, cooling_effect: false,
    package_width_mm: null, package_height_mm: null, package_length_mm: null, package_weight_g: null,
    precaution_statements: '', precautions: '', warnings: '', video_url: '',
  }
}

function ContentTextField({ id, label, value, onChange, multiline, hint }: {
  id: string; label: string; value: string; onChange: (v: string) => void; multiline?: boolean; hint?: string
}) {
  return (
    <div>
      <label htmlFor={id} className={labelClass}>{label}</label>
      {multiline ? (
        <textarea id={id} value={value} onChange={e => onChange(e.target.value)} rows={3} className={inputClass} />
      ) : (
        <input id={id} type="text" value={value} onChange={e => onChange(e.target.value)} className={inputClass} />
      )}
      {hint && <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-1">{hint}</p>}
    </div>
  )
}

function ContentNumberField({ id, label, value, onChange, unit }: {
  id: string; label: string; value: number | null | undefined; onChange: (v: number | null) => void; unit?: string
}) {
  return (
    <div>
      <label htmlFor={id} className={labelClass}>{label}{unit ? ` (${unit})` : ''}</label>
      <input
        id={id} type="number" value={value ?? ''}
        onChange={e => onChange(e.target.value === '' ? null : Number(e.target.value))}
        className={inputClass}
      />
    </div>
  )
}

function ContentCheckField({ id, label, checked, onChange }: {
  id: string; label: string; checked: boolean | null | undefined; onChange: (v: boolean) => void
}) {
  return (
    <label htmlFor={id} className="flex items-center gap-1.5 text-[11px] dark:text-gray-300 text-gray-700">
      <input id={id} type="checkbox" checked={checked ?? false} onChange={e => onChange(e.target.checked)} />
      {label}
    </label>
  )
}

export function ProductBolTab({
  productId, connection, sku, onSaveBarcode,
}: {
  productId: string
  connection: Connection
  sku: SKU | null
  // Edits the SKU's real barcode (not the Bol-only EAN override below) --
  // added here since this is where a merchant fixing a Bol validation
  // error ("missing EAN") is already looking, without having to go find
  // the Variants section. Optional: omitted when the caller has no SKU
  // update path wired (matches this component's existing sku-may-be-null
  // tolerance).
  onSaveBarcode?: (skuId: string, barcode: string) => Promise<void>
}) {
  const { token } = useAuth()

  const [, setOverride] = useState<BolProductOverride | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const [barcodeEdit, setBarcodeEdit] = useState('')
  const [barcodeHydrated, setBarcodeHydrated] = useState(false)
  const [barcodeSaving, setBarcodeSaving] = useState(false)
  const [barcodeError, setBarcodeError] = useState<string | null>(null)
  if (!barcodeHydrated && sku) {
    setBarcodeEdit(sku.barcode ?? '')
    setBarcodeHydrated(true)
  }

  const handleSaveBarcode = async () => {
    if (!sku || !onSaveBarcode) return
    setBarcodeError(null)
    setBarcodeSaving(true)
    try {
      await onSaveBarcode(sku.id, barcodeEdit)
    } catch (err) {
      setBarcodeError(err instanceof Error ? err.message : 'Failed to update barcode')
    } finally {
      setBarcodeSaving(false)
    }
  }

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

  // Product content -- separate save target from the override fields
  // above (a different endpoint, .../bol-product-content vs
  // .../bol-overrides), so it gets its own state/loading/error handling.
  const [content, setContent] = useState<UpsertBolProductContentPayload>(defaultBolProductContent())
  const [contentExpanded, setContentExpanded] = useState(false)
  const [contentLoading, setContentLoading] = useState(true)
  const [contentLoadError, setContentLoadError] = useState<string | null>(null)
  const [contentSaving, setContentSaving] = useState(false)
  const [contentSaveError, setContentSaveError] = useState<string | null>(null)
  const [contentSaved, setContentSaved] = useState(false)

  function setContentField<K extends keyof UpsertBolProductContentPayload>(key: K, value: UpsertBolProductContentPayload[K]) {
    setContent(prev => ({ ...prev, [key]: value }))
    setContentSaved(false)
  }

  // Saving the override/content fields only ever persisted them locally --
  // nothing told Bol about the change, so the live listing kept showing
  // stale data until some unrelated event (a product edit, a scheduled
  // export) happened to re-push it. Both save paths below now also push
  // to Bol immediately, same call the product page's own "Push to Bol"
  // action makes. A push failure doesn't roll back the save (the fields
  // are genuinely saved) but is surfaced separately so it's not mistaken
  // for the listing being up to date.
  const [pushing, setPushing] = useState(false)
  const [pushError, setPushError] = useState<string | null>(null)
  const [pushed, setPushed] = useState(false)

  async function pushToBol() {
    if (!token) return
    setPushError(null)
    setPushed(false)
    setPushing(true)
    try {
      await integrationsApi.pushProductToBol(token, connection.id, productId)
      setPushed(true)
    } catch (err) {
      setPushError(err instanceof Error ? err.message : 'Saved, but failed to push to Bol')
    } finally {
      setPushing(false)
    }
  }

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setContentLoading(true)
    setContentLoadError(null)
    integrationsApi.getBolProductContent(token, connection.id, productId)
      .then(result => {
        if (cancelled || !result) return
        const { product_id: _productId, connection_id: _connectionId, ...fields } = result
        setContent(prev => ({ ...prev, ...fields }))
      })
      .catch(err => { if (!cancelled) setContentLoadError(err instanceof Error ? err.message : 'Failed to load product content') })
      .finally(() => { if (!cancelled) setContentLoading(false) })
    return () => { cancelled = true }
  }, [token, connection.id, productId])

  const handleSaveContent = async () => {
    if (!token) return
    setContentSaveError(null)
    setContentSaving(true)
    try {
      const saved = await integrationsApi.upsertBolProductContent(token, connection.id, productId, content)
      const { product_id: _productId, connection_id: _connectionId, ...fields } = saved
      setContent(prev => ({ ...prev, ...fields }))
      setContentSaved(true)
      await pushToBol()
    } catch (err) {
      setContentSaveError(err instanceof Error ? err.message : 'Failed to save product content')
    } finally {
      setContentSaving(false)
    }
  }

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
      await pushToBol()
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
          {sku && (
            <div>
              <label htmlFor="bol-sku-barcode" className={labelClass}>EAN (SKU barcode)</label>
              <div className="flex items-center gap-2">
                <input
                  id="bol-sku-barcode" value={barcodeEdit} onChange={e => setBarcodeEdit(e.target.value)}
                  disabled={!onSaveBarcode} className={clsx(inputClass, 'flex-1', !onSaveBarcode && 'cursor-not-allowed opacity-70')}
                />
                {onSaveBarcode && barcodeEdit !== (sku.barcode ?? '') && (
                  <button
                    type="button" onClick={handleSaveBarcode} disabled={barcodeSaving}
                    className="px-3 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors shrink-0"
                  >
                    {barcodeSaving ? 'Saving…' : 'Save'}
                  </button>
                )}
              </div>
              <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-1">
                Required by Bol -- pushed as this offer's EAN unless overridden below.
              </p>
              {barcodeError && <p className="text-[10px] text-red-400 mt-1">{barcodeError}</p>}
            </div>
          )}
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
          <div className="flex items-center gap-2">
            <button type="button" onClick={handleSave} disabled={saving}
              className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors">
              {saving ? 'Saving…' : 'Save'}
            </button>
            {pushing && <span className="text-[10px] dark:text-gray-500 text-gray-400">Pushing to Bol…</span>}
            {!pushing && pushed && <span className="text-[10px] text-green-500">Pushed to Bol</span>}
          </div>
          {pushError && <p className="text-[11px] text-red-400">{pushError}</p>}

          <div className="pt-3 border-t dark:border-[#2D3748] border-gray-200 space-y-3">
            <button
              type="button" onClick={() => setContentExpanded(v => !v)}
              className="flex items-center justify-between w-full text-left"
            >
              <h4 className="text-xs font-semibold dark:text-gray-100 text-gray-900">
                Product content <span className="font-normal dark:text-gray-500 text-gray-400">(Bol's product-content requirements -- category, e.g. GPC Code, is set above)</span>
              </h4>
              <span className="text-[10px] dark:text-gray-500 text-gray-400">{contentExpanded ? 'Hide' : 'Show'}</span>
            </button>

            {contentExpanded && (
              <>
                {contentLoading && <p className="text-[11px] dark:text-gray-500 text-gray-400">Loading…</p>}
                {!contentLoading && contentLoadError && <p className="text-[11px] text-red-400">{contentLoadError}</p>}
                {!contentLoading && !contentLoadError && (
                  <div className="space-y-4">
                    <div>
                      <h5 className="text-[11px] font-semibold dark:text-gray-300 text-gray-600 mb-2 uppercase tracking-wider">Basic information</h5>
                      <div className="grid grid-cols-2 gap-3">
                        <ContentTextField id="bol-content-manufacturer-name" label="Manufacturer name" value={content.manufacturer_name ?? ''} onChange={v => setContentField('manufacturer_name', v)} />
                        <ContentTextField id="bol-content-manufacturer-address" label="Manufacturer address" value={content.manufacturer_address ?? ''} onChange={v => setContentField('manufacturer_address', v)} />
                        <ContentTextField id="bol-content-manufacturer-contact" label="Manufacturer email / contact form" value={content.manufacturer_contact ?? ''} onChange={v => setContentField('manufacturer_contact', v)} />
                        <ContentTextField id="bol-content-shelf-life-closed" label="Shelf life (unopened)" value={content.shelf_life_closed ?? ''} onChange={v => setContentField('shelf_life_closed', v)} hint="e.g. 'More than 30 months'" />
                        <ContentTextField id="bol-content-shelf-life-opened" label="Shelf life after opening" value={content.shelf_life_after_opening ?? ''} onChange={v => setContentField('shelf_life_after_opening', v)} hint="e.g. '1M' (month)" />
                        <ContentTextField id="bol-content-country-of-origin" label="Country of origin" value={content.country_of_origin ?? ''} onChange={v => setContentField('country_of_origin', v)} />
                        <ContentTextField id="bol-content-substance" label="Substance" value={content.substance ?? ''} onChange={v => setContentField('substance', v)} hint="e.g. Cream, Gel, Liquid" />
                        <ContentTextField id="bol-content-instruction-language" label="Instruction manual language" value={content.instruction_language ?? ''} onChange={v => setContentField('instruction_language', v)} />
                        <ContentTextField id="bol-content-packaging-language" label="Packaging language" value={content.packaging_language ?? ''} onChange={v => setContentField('packaging_language', v)} />
                        <div className="grid grid-cols-2 gap-2">
                          <ContentTextField id="bol-content-content-amount" label="Content amount" value={content.content_amount ?? ''} onChange={v => setContentField('content_amount', v)} />
                          <ContentTextField id="bol-content-content-unit" label="Content unit" value={content.content_unit ?? ''} onChange={v => setContentField('content_unit', v)} hint="e.g. g, ml" />
                        </div>
                      </div>
                      <div className="mt-3">
                        <ContentTextField id="bol-content-ingredients" label="Ingredients" value={content.ingredients ?? ''} onChange={v => setContentField('ingredients', v)} multiline />
                      </div>
                    </div>

                    <div>
                      <h5 className="text-[11px] font-semibold dark:text-gray-300 text-gray-600 mb-2 uppercase tracking-wider">Optional</h5>
                      <div className="grid grid-cols-2 gap-3">
                        <ContentTextField id="bol-content-brand" label="Brand" value={content.brand ?? ''} onChange={v => setContentField('brand', v)} />
                        <ContentNumberField id="bol-content-items-per-package" label="Items per package" value={content.items_per_package} onChange={v => setContentField('items_per_package', v)} />
                        <ContentTextField id="bol-content-package-insert-url" label="Package insert (URL)" value={content.package_insert_url ?? ''} onChange={v => setContentField('package_insert_url', v)} />
                        <ContentTextField id="bol-content-skin-type" label="Skin type" value={content.skin_type ?? ''} onChange={v => setContentField('skin_type', v)} />
                        <ContentTextField id="bol-content-hazard-statements" label="Hazard statements (H-statements)" value={content.hazard_statements ?? ''} onChange={v => setContentField('hazard_statements', v)} />
                        <ContentTextField id="bol-content-precaution-statements" label="Precaution statements (P-statements)" value={content.precaution_statements ?? ''} onChange={v => setContentField('precaution_statements', v)} />
                        <div className="grid grid-cols-2 gap-2">
                          <ContentTextField id="bol-content-dispenser-amount" label="Content per dispenser" value={content.content_per_dispenser_amount ?? ''} onChange={v => setContentField('content_per_dispenser_amount', v)} />
                          <ContentTextField id="bol-content-dispenser-unit" label="Dispenser unit" value={content.content_per_dispenser_unit ?? ''} onChange={v => setContentField('content_per_dispenser_unit', v)} />
                        </div>
                        <ContentNumberField id="bol-content-package-width" label="Package width" value={content.package_width_mm} onChange={v => setContentField('package_width_mm', v)} unit="mm" />
                        <ContentNumberField id="bol-content-package-height" label="Package height" value={content.package_height_mm} onChange={v => setContentField('package_height_mm', v)} unit="mm" />
                        <ContentNumberField id="bol-content-package-length" label="Package length" value={content.package_length_mm} onChange={v => setContentField('package_length_mm', v)} unit="mm" />
                        <ContentNumberField id="bol-content-package-weight" label="Package weight" value={content.package_weight_g} onChange={v => setContentField('package_weight_g', v)} unit="g" />
                        <ContentTextField id="bol-content-video-url" label="Video (URL)" value={content.video_url ?? ''} onChange={v => setContentField('video_url', v)} />
                      </div>
                      <div className="grid grid-cols-2 gap-3 mt-3">
                        <ContentTextField id="bol-content-precautions" label="Precautions" value={content.precautions ?? ''} onChange={v => setContentField('precautions', v)} multiline />
                        <ContentTextField id="bol-content-warnings" label="Warnings" value={content.warnings ?? ''} onChange={v => setContentField('warnings', v)} multiline />
                      </div>
                      <div className="grid grid-cols-3 gap-2 mt-3">
                        <ContentCheckField id="bol-content-is-gift" label="Gift" checked={content.is_gift} onChange={v => setContentField('is_gift', v)} />
                        <ContentCheckField id="bol-content-dermatologically-tested" label="Dermatologically tested" checked={content.dermatologically_tested} onChange={v => setContentField('dermatologically_tested', v)} />
                        <ContentCheckField id="bol-content-hypoallergenic" label="Hypoallergenic" checked={content.hypoallergenic} onChange={v => setContentField('hypoallergenic', v)} />
                        <ContentCheckField id="bol-content-contains-zinc-oxide" label="Contains zinc oxide" checked={content.contains_zinc_oxide} onChange={v => setContentField('contains_zinc_oxide', v)} />
                        <ContentCheckField id="bol-content-fragrance-free" label="Fragrance-free" checked={content.fragrance_free} onChange={v => setContentField('fragrance_free', v)} />
                        <ContentCheckField id="bol-content-ph-neutral" label="pH-neutral" checked={content.ph_neutral} onChange={v => setContentField('ph_neutral', v)} />
                        <ContentCheckField id="bol-content-cooling-effect" label="Cooling effect" checked={content.cooling_effect} onChange={v => setContentField('cooling_effect', v)} />
                      </div>
                    </div>

                    {contentSaveError && <p className="text-[11px] text-red-400">{contentSaveError}</p>}
                    <div className="flex items-center gap-2">
                      <button type="button" onClick={handleSaveContent} disabled={contentSaving}
                        className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors">
                        {contentSaving ? 'Saving…' : 'Save product content'}
                      </button>
                      {contentSaved && <span className="text-[10px] text-green-500">Saved</span>}
                      {pushing && <span className="text-[10px] dark:text-gray-500 text-gray-400">Pushing to Bol…</span>}
                      {!pushing && pushed && <span className="text-[10px] text-green-500">Pushed to Bol</span>}
                    </div>
                    {pushError && <p className="text-[11px] text-red-400">{pushError}</p>}
                  </div>
                )}
              </>
            )}
          </div>

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
