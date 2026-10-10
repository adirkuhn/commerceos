import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Package } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import * as integrationsApi from '../integrations/api'
import type { Connection, BolProductOverride, BolCommission } from '../integrations/types'
import type { UpsertBolProductContentPayload } from '../integrations/api'
import type { SKU } from './types'
import { skinTypeOptions, shelfLifeClosedOptions, shelfLifeAfterOpeningOptions, countryOfOriginOptions, hazardStatementsOptions, substanceOptions, precautionStatementsOptions } from './bolContentLovs'
import { ContentCheckField, ContentNumberField, ContentSelectField, ContentTextField, defaultBolProductContent, inputClass, labelClass } from './BolContentFields'
import { apiErrorText } from '../api/apiErrorText'

// Brand name, not translated.
const BRAND = 'Bol.com'

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

export function ProductBolTab({
  productId, connection, sku, productName, onSaveBarcode,
}: {
  productId: string
  connection: Connection
  sku: SKU | null
  // Used only as the title-override field's placeholder (what Bol would
  // sync without an override) -- never sent anywhere itself.
  productName?: string
  // Edits the SKU's real barcode (not the Bol-only EAN override below) --
  // added here since this is where a merchant fixing a Bol validation
  // error ("missing EAN") is already looking, without having to go find
  // the Variants section. Optional: omitted when the caller has no SKU
  // update path wired (matches this component's existing sku-may-be-null
  // tolerance).
  onSaveBarcode?: (skuId: string, barcode: string) => Promise<void>
}) {
  const { token } = useAuth()
  const { t } = useTranslation(['pms', 'common'])

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
      setBarcodeError(apiErrorText(err) ?? t('bol.barcodeFailed'))
    } finally {
      setBarcodeSaving(false)
    }
  }

  const [nameOverride, setNameOverride] = useState('')
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

  // Bol's own "why isn't this offer for sale" reason, checked server-side
  // after every push (GET .../not-for-sale-reasons) -- surfaced here since
  // a push can succeed with no error at all while Bol still refuses to
  // list the offer (e.g. a still-missing mandatory content attribute).
  const [notForSaleNote, setNotForSaleNote] = useState<string | null>(null)
  useEffect(() => {
    if (!token) return
    let cancelled = false
    integrationsApi.getProductChannelListing(token, connection.id, productId)
      .then(result => { if (!cancelled) setNotForSaleNote(result?.not_for_sale_note || null) })
      .catch(() => { /* non-critical: the rest of the tab still works without this */ })
    return () => { cancelled = true }
  }, [token, connection.id, productId])

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
      .catch(err => { if (!cancelled) setContentLoadError(apiErrorText(err) ?? t('bol.contentLoadFailed')) })
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
    } catch (err) {
      setContentSaveError(apiErrorText(err) ?? t('bol.contentSaveFailed'))
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
          setNameOverride(result.name_override ?? '')
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
      .catch(err => { if (!cancelled) setLoadError(apiErrorText(err) ?? t('bol.loadFailed')) })
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
      .catch(err => { if (!cancelled) setCommissionError(apiErrorText(err) ?? t('bol.commissionFailed')) })
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
        nameOverride: nameOverride || null,
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
      setSaveError(apiErrorText(err) ?? t('bol.saveFailed'))
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
    <div className="rounded-xl border border-line p-4 space-y-4">
      <div className="flex items-center gap-2">
        <Package size={13} className="text-ink-3" />
        <h3 className="text-xs font-semibold text-ink">{BRAND}</h3>
      </div>

      {notForSaleNote && (
        <div className="rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2">
          <p className="text-2xs font-medium text-warning">{t('bol.notForSale')}</p>
          <p className="text-2xs dark:text-amber-300/90 text-warning mt-0.5">{notForSaleNote}</p>
        </div>
      )}

      {loading && <p className="text-2xs text-ink-4">{t('common:status.loading')}</p>}
      {!loading && loadError && <p className="text-2xs text-danger">{loadError}</p>}

      {!loading && !loadError && (
        <>
          {sku && (
            <div>
              <label htmlFor="bol-sku-barcode" className={labelClass}>{t('bol.ean')}</label>
              <div className="flex items-center gap-2">
                <input
                  id="bol-sku-barcode" value={barcodeEdit} onChange={e => setBarcodeEdit(e.target.value)}
                  disabled={!onSaveBarcode} className={clsx(inputClass, 'flex-1', !onSaveBarcode && 'cursor-not-allowed opacity-70')}
                />
                {onSaveBarcode && barcodeEdit !== (sku.barcode ?? '') && (
                  <button
                    type="button" onClick={handleSaveBarcode} disabled={barcodeSaving}
                    className="px-3 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors shrink-0"
                  >
                    {barcodeSaving ? t('common:actions.saving') : t('common:actions.save')}
                  </button>
                )}
              </div>
              <p className="text-2xs text-ink-4 mt-1">
                {t('bol.eanHint')}
              </p>
              {barcodeError && <p className="text-2xs text-danger mt-1">{barcodeError}</p>}
            </div>
          )}
          <div>
            <label htmlFor="bol-name-override" className={labelClass}>{t('bol.titleOverride')}</label>
            <input id="bol-name-override" value={nameOverride} onChange={e => setNameOverride(e.target.value)}
              placeholder={productName} className={inputClass} />
            <p className="text-2xs text-ink-4 mt-1">
              {t('bol.titleOverrideHint')}
            </p>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label htmlFor="bol-ean-override" className={labelClass}>{t('bol.eanOverride')}</label>
              <input id="bol-ean-override" value={eanOverride} onChange={e => setEanOverride(e.target.value)}
                placeholder={sku?.barcode ?? ''} className={inputClass} />
            </div>
            <div>
              <label htmlFor="bol-gpc-code" className={labelClass}>{t('bol.gpcCode')}</label>
              <input id="bol-gpc-code" value={gpcCode} onChange={e => setGpcCode(e.target.value)} className={inputClass} />
              <p className="text-2xs text-ink-4 mt-1">{t('bol.gpcHint')}</p>
            </div>
            <div>
              <label htmlFor="bol-condition" className={labelClass}>{t('bol.condition')}</label>
              <select id="bol-condition" value={condition} onChange={e => setCondition(e.target.value as BolProductOverride['condition_category'])} className={inputClass}>
                <option value="NEW">{t('bol.conditionNew')}</option>
                <option value="SECONDHAND">{t('bol.conditionSecondhand')}</option>
                <option value="REFURBISHED">{t('bol.conditionRefurbished')}</option>
              </select>
            </div>
            <div>
              <label className={labelClass}>{t('bol.fulfilment')}</label>
              <input disabled readOnly value={t('bol.fbr')} className={`${inputClass} cursor-not-allowed opacity-70`} />
            </div>
            <div>
              <label htmlFor="bol-reference" className={labelClass}>{t('bol.reference')}</label>
              <input id="bol-reference" value={referenceOverride} onChange={e => setReferenceOverride(e.target.value)} className={inputClass} />
            </div>
            <div>
              <label htmlFor="bol-custom-price" className={labelClass}>{t('bol.customPriceOptional')}</label>
              <input id="bol-custom-price" value={customPrice} onChange={e => setCustomPrice(e.target.value)}
                placeholder={fmt(defaultPrice)} className={inputClass} />
              <p className="text-2xs text-ink-4 mt-1">{t('bol.customPriceHint', { price: fmt(defaultPrice) })}</p>
            </div>
          </div>

          {saveError && <p className="text-2xs text-danger">{saveError}</p>}
          <div className="flex items-center gap-2">
            <button type="button" onClick={handleSave} disabled={saving}
              className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors">
              {saving ? t('common:actions.saving') : t('common:actions.save')}
            </button>
          </div>

          <div className="pt-3 border-t border-line space-y-3">
            <button
              type="button" onClick={() => setContentExpanded(v => !v)}
              className="flex items-center justify-between w-full text-left"
            >
              <h4 className="text-xs font-semibold text-ink">
                {t('bol.contentTitle')} <span className="font-normal text-ink-4">{t('bol.contentSubtitle')}</span>
              </h4>
              <span className="text-2xs text-ink-4">{contentExpanded ? t('bol.hide') : t('bol.show')}</span>
            </button>

            {contentExpanded && (
              <>
                {contentLoading && <p className="text-2xs text-ink-4">{t('common:status.loading')}</p>}
                {!contentLoading && contentLoadError && <p className="text-2xs text-danger">{contentLoadError}</p>}
                {!contentLoading && !contentLoadError && (
                  <div className="space-y-4">
                    <div>
                      <h5 className="text-2xs font-semibold text-ink-2 mb-2 uppercase tracking-wider">{t('bol.basicInfo')}</h5>
                      <div className="grid grid-cols-2 gap-3">
                        <ContentTextField id="bol-content-manufacturer-name" label={t('bol.manufacturerName')} value={content.manufacturer_name ?? ''} onChange={v => setContentField('manufacturer_name', v)} />
                        <ContentTextField id="bol-content-manufacturer-address" label={t('bol.manufacturerAddress')} value={content.manufacturer_address ?? ''} onChange={v => setContentField('manufacturer_address', v)} />
                        <ContentTextField id="bol-content-manufacturer-contact" label={t('bol.manufacturerContact')} value={content.manufacturer_contact ?? ''} onChange={v => setContentField('manufacturer_contact', v)} />
                        <ContentSelectField id="bol-content-shelf-life-closed" label={t('bol.shelfLifeClosed')} value={content.shelf_life_closed ?? ''} onChange={v => setContentField('shelf_life_closed', v)} options={shelfLifeClosedOptions} />
                        <ContentSelectField id="bol-content-shelf-life-opened" label={t('bol.shelfLifeOpened')} value={content.shelf_life_after_opening ?? ''} onChange={v => setContentField('shelf_life_after_opening', v)} options={shelfLifeAfterOpeningOptions} />
                        <ContentSelectField id="bol-content-country-of-origin" label={t('bol.countryOfOrigin')} value={content.country_of_origin ?? ''} onChange={v => setContentField('country_of_origin', v)} options={countryOfOriginOptions} />
                        <ContentSelectField id="bol-content-substance" label={t('bol.substance')} value={content.substance ?? ''} onChange={v => setContentField('substance', v)} options={substanceOptions} />
                        <ContentTextField id="bol-content-instruction-language" label={t('bol.instructionLanguage')} value={content.instruction_language ?? ''} onChange={v => setContentField('instruction_language', v)} />
                        <ContentTextField id="bol-content-packaging-language" label={t('bol.packagingLanguage')} value={content.packaging_language ?? ''} onChange={v => setContentField('packaging_language', v)} />
                        <div className="grid grid-cols-2 gap-2">
                          <ContentTextField id="bol-content-content-amount" label={t('bol.contentAmount')} value={content.content_amount ?? ''} onChange={v => setContentField('content_amount', v)} />
                          <ContentTextField id="bol-content-content-unit" label={t('bol.contentUnit')} value={content.content_unit ?? ''} onChange={v => setContentField('content_unit', v)} hint={t('bol.contentUnitHint')} />
                        </div>
                      </div>
                      <div className="mt-3">
                        <ContentTextField id="bol-content-ingredients" label={t('bol.ingredients')} value={content.ingredients ?? ''} onChange={v => setContentField('ingredients', v)} multiline />
                      </div>
                    </div>

                    <div>
                      <h5 className="text-2xs font-semibold text-ink-2 mb-2 uppercase tracking-wider">{t('bol.optional')}</h5>
                      <div className="grid grid-cols-2 gap-3">
                        <ContentTextField id="bol-content-brand" label={t('bol.brand')} value={content.brand ?? ''} onChange={v => setContentField('brand', v)} />
                        <ContentNumberField id="bol-content-items-per-package" label={t('bol.itemsPerPackage')} value={content.items_per_package} onChange={v => setContentField('items_per_package', v)} />
                        <ContentTextField id="bol-content-package-insert-url" label={t('bol.packageInsertUrl')} value={content.package_insert_url ?? ''} onChange={v => setContentField('package_insert_url', v)} />
                        <ContentSelectField id="bol-content-skin-type" label={t('bol.skinType')} value={content.skin_type ?? ''} onChange={v => setContentField('skin_type', v)} options={skinTypeOptions} />
                        <ContentSelectField id="bol-content-hazard-statements" label={t('bol.hazardStatements')} value={content.hazard_statements ?? ''} onChange={v => setContentField('hazard_statements', v)} options={hazardStatementsOptions} hint={t('bol.hazardHint')} />
                        <ContentSelectField id="bol-content-precaution-statements" label={t('bol.precautionStatements')} value={content.precaution_statements ?? ''} onChange={v => setContentField('precaution_statements', v)} options={precautionStatementsOptions} hint={t('bol.precautionHint')} />
                        <div className="grid grid-cols-2 gap-2">
                          <ContentTextField id="bol-content-dispenser-amount" label={t('bol.dispenserAmount')} value={content.content_per_dispenser_amount ?? ''} onChange={v => setContentField('content_per_dispenser_amount', v)} />
                          <ContentTextField id="bol-content-dispenser-unit" label={t('bol.dispenserUnit')} value={content.content_per_dispenser_unit ?? ''} onChange={v => setContentField('content_per_dispenser_unit', v)} />
                        </div>
                        <ContentNumberField id="bol-content-package-width" label={t('bol.packageWidth')} value={content.package_width_mm} onChange={v => setContentField('package_width_mm', v)} unit="mm" />
                        <ContentNumberField id="bol-content-package-height" label={t('bol.packageHeight')} value={content.package_height_mm} onChange={v => setContentField('package_height_mm', v)} unit="mm" />
                        <ContentNumberField id="bol-content-package-length" label={t('bol.packageLength')} value={content.package_length_mm} onChange={v => setContentField('package_length_mm', v)} unit="mm" />
                        <ContentNumberField id="bol-content-package-weight" label={t('bol.packageWeight')} value={content.package_weight_g} onChange={v => setContentField('package_weight_g', v)} unit="g" />
                        <ContentTextField id="bol-content-video-url" label={t('bol.videoUrl')} value={content.video_url ?? ''} onChange={v => setContentField('video_url', v)} />
                      </div>
                      <div className="grid grid-cols-2 gap-3 mt-3">
                        <ContentTextField id="bol-content-precautions" label={t('bol.precautions')} value={content.precautions ?? ''} onChange={v => setContentField('precautions', v)} multiline />
                        <ContentTextField id="bol-content-warnings" label={t('bol.warnings')} value={content.warnings ?? ''} onChange={v => setContentField('warnings', v)} multiline />
                      </div>
                      <div className="grid grid-cols-3 gap-2 mt-3">
                        <ContentCheckField id="bol-content-is-gift" label={t('bol.gift')} checked={content.is_gift} onChange={v => setContentField('is_gift', v)} />
                        <ContentCheckField id="bol-content-dermatologically-tested" label={t('bol.dermatologicallyTested')} checked={content.dermatologically_tested} onChange={v => setContentField('dermatologically_tested', v)} />
                        <ContentCheckField id="bol-content-hypoallergenic" label={t('bol.hypoallergenic')} checked={content.hypoallergenic} onChange={v => setContentField('hypoallergenic', v)} />
                        <ContentCheckField id="bol-content-contains-zinc-oxide" label={t('bol.zincOxide')} checked={content.contains_zinc_oxide} onChange={v => setContentField('contains_zinc_oxide', v)} />
                        <ContentCheckField id="bol-content-fragrance-free" label={t('bol.fragranceFree')} checked={content.fragrance_free} onChange={v => setContentField('fragrance_free', v)} />
                        <ContentCheckField id="bol-content-ph-neutral" label={t('bol.phNeutral')} checked={content.ph_neutral} onChange={v => setContentField('ph_neutral', v)} />
                        <ContentCheckField id="bol-content-cooling-effect" label={t('bol.coolingEffect')} checked={content.cooling_effect} onChange={v => setContentField('cooling_effect', v)} />
                      </div>
                    </div>

                    {contentSaveError && <p className="text-2xs text-danger">{contentSaveError}</p>}
                    <div className="flex items-center gap-2">
                      <button type="button" onClick={handleSaveContent} disabled={contentSaving}
                        className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors">
                        {contentSaving ? t('common:actions.saving') : t('bol.saveContent')}
                      </button>
                      {contentSaved && <span className="text-2xs text-success">{t('common:status.saved')}</span>}
                    </div>
                  </div>
                )}
              </>
            )}
          </div>

          <div className="pt-3 border-t border-line space-y-3">
            <h4 className="text-xs font-semibold text-ink">{t('bol.profitCalculator')}</h4>
            {commissionError && <p className="text-2xs text-danger">{commissionError}</p>}

            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className={labelClass}>{t('bol.sellPrice')}</label>
                <input disabled readOnly value={fmt(sellPrice)} className={`${inputClass} cursor-not-allowed opacity-70`} />
                {customPrice.trim() !== '' && <p className="text-2xs text-accent mt-1">{t('bol.customPrice')}</p>}
              </div>
              <div>
                <label className={labelClass}>{t('bol.costPrice')}</label>
                <input disabled readOnly value={fmt(costPrice)} className={`${inputClass} cursor-not-allowed opacity-70`} />
              </div>
              <div>
                <label htmlFor="bol-tax-rate" className={labelClass}>{t('bol.taxRate')}</label>
                <input id="bol-tax-rate" value={taxRatePercent} onChange={e => setTaxRatePercent(e.target.value)} className={inputClass} />
              </div>
              <div>
                <label htmlFor="bol-commission-percent" className={labelClass}>
                  {t('bol.commission')} {commission && !commissionOverridden && t('bol.live')}
                </label>
                <input id="bol-commission-percent" value={commissionPercentInput}
                  onChange={e => { setCommissionPercentInput(e.target.value); setCommissionOverridden(true) }}
                  className={inputClass} />
                {commissionOverridden && (
                  <button type="button" className="text-2xs text-accent mt-1"
                    onClick={() => { setCommissionOverridden(false); if (commission) setCommissionPercentInput(fmt(commission.percentage)) }}>
                    {t('bol.useLive')}
                  </button>
                )}
              </div>
              <div>
                <label className={labelClass}>{t('bol.fixedFee')}</label>
                <input disabled readOnly value={fmt(commission?.fixed_amount ?? 0)} className={`${inputClass} cursor-not-allowed opacity-70`} />
              </div>
              <div>
                <label htmlFor="bol-shipping-cost" className={labelClass}>{t('bol.shippingCost')}</label>
                <input id="bol-shipping-cost" value={shippingCost} onChange={e => setShippingCost(e.target.value)} className={inputClass} />
              </div>
            </div>

            <div className="rounded-lg bg-sunken p-3 space-y-1 text-2xs">
              <div className="flex justify-between text-ink-3"><span>{t('bol.sellPrice')}</span><span>{fmt(sellPrice)}</span></div>
              <div className="flex justify-between text-ink-3"><span>{t('bol.minusCost')}</span><span>{fmt(costPrice)}</span></div>
              <div className="flex justify-between text-ink-3"><span>{t('bol.minusTax')}</span><span>{fmt(result.taxAmount)}</span></div>
              <div className="flex justify-between text-ink-3"><span>{t('bol.minusCommission')}</span><span>{fmt(result.commissionAmount)}</span></div>
              <div className="flex justify-between text-ink-3"><span>{t('bol.minusShipping')}</span><span>{fmt(parsedShipping)}</span></div>
              <div className="flex justify-between font-semibold text-ink pt-1 border-t border-line">
                <span>{t('bol.profit')}</span><span>{fmt(result.profit)}</span>
              </div>
              <div className="flex justify-between text-ink-3">
                <span>{t('bol.margin')}</span><span>{fmt(result.marginPercent)}%</span>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
