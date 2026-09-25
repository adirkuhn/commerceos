import { useState } from 'react'
import { Package, Trash2, Edit3, X, ChevronDown, ChevronRight } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useProductDetail } from './hooks/useProductDetail'
import { BulkAddWeightPortionsModal } from './BulkAddWeightPortionsModal'
import type { SKUStatus, SKUUnitType, SKUPricingMode, VariantWithSKUs, Media } from './types'
import { useTaxClasses } from '../tax/hooks/useTaxClasses'
import { useTaxRates } from '../tax/hooks/useTaxRates'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import { useConfirm } from '../components/common/useConfirm'
import { CURRENCIES, SKU_STATUS_STYLES, formatMinorUnitsAsInput, formatPrice, normalizeScaleCodeInput, parseIntOrUndefined, parseOptionalPriceToMinorUnits, parsePriceToMinorUnits } from './productFormat'
import { CaseOfSKUPicker, StockBadge } from './VariantsSection'
import { BarcodeAliasesEditor, SKUCurrencyPricesEditor, ScaleCodeAliasesEditor } from './SkuAliasEditors'
import { MarginInsightCard } from './ProductPricingCards'

interface VariantRowProps {
  variant: VariantWithSKUs
  detail: ReturnType<typeof useProductDetail>
  productMedia: Media[]
}

export function VariantRow({ variant, detail, productMedia }: VariantRowProps) {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { settings: tenantSettings } = useTenantSettings(token ?? '')
  // Same "resolve tax class -> country rate" lookup as ProductEditPage's
  // taxRateForPreview, but scoped to this row's own SKU.tax_class_id
  // instead of the product-level field -- see SalesOrders.tsx's lineTax
  // for the same per-entity-override-with-tenant-fallback pattern.
  const { classes: taxClasses } = useTaxClasses(token ?? '')
  const { rates: taxRates } = useTaxRates(token ?? '')
  const [addingSKU, setAddingSKU] = useState(false)
  const [showMargin, setShowMargin] = useState(false)
  const [imageError, setImageError] = useState('')
  const [skuCode, setSkuCode] = useState('')
  const [barcode, setBarcode] = useState('')
  const [price, setPrice] = useState('')
  const [salePrice, setSalePrice] = useState('')
  const [saleEndDate, setSaleEndDate] = useState('')
  const [currency, setCurrency] = useState('USD')
  const [status, setStatus] = useState<SKUStatus>('active')
  const [isVirtual, setIsVirtual] = useState(false)
  const [weightGrams, setWeightGrams] = useState('')
  const [lengthMM, setLengthMM] = useState('')
  const [widthMM, setWidthMM] = useState('')
  const [heightMM, setHeightMM] = useState('')
  // Weight pool (see pms.SKU.UnitType/PortionedFromSKUID) -- a SKU is
  // either a weight pool itself (unit_type kilo), or portioned from one,
  // never both.
  const [unitType, setUnitType] = useState<SKUUnitType>('unit')
  const [portionedFromSKUID, setPortionedFromSKUID] = useState<string | null>(null)
  const [portionedFromLabel, setPortionedFromLabel] = useState('')
  const [portionPickerOpen, setPortionPickerOpen] = useState(false)
  // See ProductEditPage's identical fields for the weight-embedded
  // (butcher/deli scale) barcode -- same convention here.
  const [scaleCode, setScaleCode] = useState('')
  const [pricingMode, setPricingMode] = useState<SKUPricingMode>('fixed')
  const [pricePerKilo, setPricePerKilo] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  // See ProductEditPage's identical currency-defaulting comment -- same
  // "wait for tenantSettings to actually arrive" reasoning.
  const [currencyDefaulted, setCurrencyDefaulted] = useState(false)
  if (!currencyDefaulted && tenantSettings) {
    if (tenantSettings.default_currency) setCurrency(tenantSettings.default_currency)
    setCurrencyDefaulted(true)
  }

  const fieldLabelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wide mb-0.5'
  const fieldInputClass = 'w-full px-2 py-1 text-2xs rounded bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'

  const handleAddSKU = async () => {
    setError('')
    setSubmitting(true)
    try {
      await detail.addSKU(variant.id, {
        sku_code: skuCode,
        barcode: barcode || undefined,
        price_minor_units: parsePriceToMinorUnits(price),
        sale_price_minor_units: parseOptionalPriceToMinorUnits(salePrice),
        sale_end_date: saleEndDate || null,
        currency_code: currency,
        is_virtual: isVirtual,
        weight_grams: isVirtual ? undefined : parseIntOrUndefined(weightGrams),
        length_mm: isVirtual ? undefined : parseIntOrUndefined(lengthMM),
        width_mm: isVirtual ? undefined : parseIntOrUndefined(widthMM),
        height_mm: isVirtual ? undefined : parseIntOrUndefined(heightMM),
        unit_type: unitType,
        portioned_from_sku_id: portionedFromSKUID,
        scale_code: scaleCode || null,
        pricing_mode: pricingMode,
        price_per_kilo_minor_units: pricingMode === 'per_weight' ? parseOptionalPriceToMinorUnits(pricePerKilo) : null,
        status,
      })
      setAddingSKU(false)
      setSkuCode(''); setBarcode(''); setPrice(''); setSalePrice(''); setSaleEndDate('')
      setCurrency(tenantSettings?.default_currency || 'USD'); setStatus('active')
      setIsVirtual(false); setWeightGrams(''); setLengthMM(''); setWidthMM(''); setHeightMM('')
      setUnitType('unit'); setPortionedFromSKUID(null); setPortionedFromLabel('')
      setScaleCode(''); setPricingMode('fixed'); setPricePerKilo('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add SKU')
    } finally {
      setSubmitting(false)
    }
  }

  const sku = variant.skus?.[0]
  const defaultTaxClassId = taxClasses.find(c => c.is_default)?.id ?? null
  const effectiveTaxClassId = sku?.tax_class_id ?? defaultTaxClassId
  const taxRateForSku = effectiveTaxClassId && tenantSettings?.default_tax_country
    ? taxRates.find(r => r.tax_class_id === effectiveTaxClassId && r.country === tenantSettings.default_tax_country)
    : undefined
  const [bulkSaving, setBulkSaving] = useState(false)
  const [bulkError, setBulkError] = useState('')
  const [existingPortionPickerOpen, setExistingPortionPickerOpen] = useState(false)
  const [bulkAddOpen, setBulkAddOpen] = useState(false)

  // Once a variant's SKU exists, its barcode had no display or edit path
  // at all in this component -- only settable in the "add SKU" form below,
  // never again after creation. That's a real gap: a per-SKU integration
  // (e.g. Bol, which treats SKU.Barcode as the EAN) can only ever be
  // fixed by deleting and recreating the SKU. Edits here save immediately,
  // same "no batched Save step" convention as saveBulkPool below.
  const [barcodeEdit, setBarcodeEdit] = useState('')
  const [barcodeHydrated, setBarcodeHydrated] = useState(false)
  const [barcodeSaving, setBarcodeSaving] = useState(false)
  const [barcodeError, setBarcodeError] = useState('')
  if (!barcodeHydrated && sku) {
    setBarcodeEdit(sku.barcode ?? '')
    setBarcodeHydrated(true)
  }

  const saveBarcode = async () => {
    if (!sku) return
    setBarcodeError('')
    setBarcodeSaving(true)
    try {
      await detail.updateSKU(sku.id, {
        sku_code: sku.sku_code,
        barcode: barcodeEdit || undefined,
        price_minor_units: sku.price_minor_units,
        sale_price_minor_units: sku.sale_price_minor_units,
        sale_end_date: sku.sale_end_date,
        currency_code: sku.currency_code,
        weight_grams: sku.weight_grams,
        length_mm: sku.length_mm,
        width_mm: sku.width_mm,
        height_mm: sku.height_mm,
        is_virtual: sku.is_virtual,
        tracks_expiry: sku.tracks_expiry,
        case_of_sku_id: sku.case_of_sku_id,
        units_per_case: sku.units_per_case,
        unit_type: sku.unit_type,
        portioned_from_sku_id: sku.portioned_from_sku_id,
        scale_code: sku.scale_code,
        pricing_mode: sku.pricing_mode,
        price_per_kilo_minor_units: sku.price_per_kilo_minor_units,
        status: sku.status,
      })
    } catch (err) {
      setBarcodeError(err instanceof Error ? err.message : 'Failed to update barcode')
    } finally {
      setBarcodeSaving(false)
    }
  }

  // Editing an existing SKU's bulk-pool link saves immediately (matches
  // handleSelectImage's pattern in this component) -- there's no batched
  // "Save" step for an already-created variant's SKU here, unlike the flat
  // single-SKU form on the product's General tab.
  const saveBulkPool = async (next: { unitType: SKUUnitType; portionedFromSkuId: string | null }) => {
    if (!sku) return
    setBulkError('')
    setBulkSaving(true)
    try {
      await detail.updateSKU(sku.id, {
        sku_code: sku.sku_code,
        barcode: sku.barcode ?? undefined,
        price_minor_units: sku.price_minor_units,
        sale_price_minor_units: sku.sale_price_minor_units,
        sale_end_date: sku.sale_end_date,
        currency_code: sku.currency_code,
        weight_grams: sku.weight_grams,
        length_mm: sku.length_mm,
        width_mm: sku.width_mm,
        height_mm: sku.height_mm,
        is_virtual: sku.is_virtual,
        tracks_expiry: sku.tracks_expiry,
        case_of_sku_id: sku.case_of_sku_id,
        units_per_case: sku.units_per_case,
        unit_type: next.unitType,
        portioned_from_sku_id: next.portionedFromSkuId,
        scale_code: sku.scale_code,
        pricing_mode: sku.pricing_mode,
        price_per_kilo_minor_units: sku.price_per_kilo_minor_units,
        status: sku.status,
      })
      setExistingPortionPickerOpen(false)
    } catch (err) {
      setBulkError(err instanceof Error ? err.message : 'Failed to update weight pool link')
    } finally {
      setBulkSaving(false)
    }
  }
  // Editing an existing variant/SKU's own fields (name, price, status,
  // weight/dimensions, etc.) -- everything the "Add SKU" form below covers
  // except barcode and the weight-pool link, which already have their own
  // dedicated inline-edit controls (saveBarcode/saveBulkPool above); adding
  // a second path to change those here would just create two conflicting
  // places to do the same thing. Saves both the variant (name) and the SKU
  // (everything else) as two separate calls -- they're different entities/
  // endpoints already, same as every other write in this component.
  const [editing, setEditing] = useState(false)
  const [editName, setEditName] = useState('')
  const [editSkuCode, setEditSkuCode] = useState('')
  const [editPrice, setEditPrice] = useState('')
  const [editSalePrice, setEditSalePrice] = useState('')
  const [editSaleEndDate, setEditSaleEndDate] = useState('')
  const [editCurrency, setEditCurrency] = useState('USD')
  const [editStatus, setEditStatus] = useState<SKUStatus>('active')
  const [editIsVirtual, setEditIsVirtual] = useState(false)
  const [editWeightGrams, setEditWeightGrams] = useState('')
  const [editLengthMM, setEditLengthMM] = useState('')
  const [editWidthMM, setEditWidthMM] = useState('')
  const [editHeightMM, setEditHeightMM] = useState('')
  const [editError, setEditError] = useState('')
  const [editSaving, setEditSaving] = useState(false)

  const openEdit = () => {
    setEditError('')
    setEditName(variant.name)
    if (sku) {
      setEditSkuCode(sku.sku_code)
      setEditPrice(formatMinorUnitsAsInput(sku.price_minor_units))
      setEditSalePrice(sku.sale_price_minor_units != null ? formatMinorUnitsAsInput(sku.sale_price_minor_units) : '')
      setEditSaleEndDate(sku.sale_end_date ?? '')
      setEditCurrency(sku.currency_code)
      setEditStatus(sku.status)
      setEditIsVirtual(sku.is_virtual)
      setEditWeightGrams(sku.weight_grams != null ? String(sku.weight_grams) : '')
      setEditLengthMM(sku.length_mm != null ? String(sku.length_mm) : '')
      setEditWidthMM(sku.width_mm != null ? String(sku.width_mm) : '')
      setEditHeightMM(sku.height_mm != null ? String(sku.height_mm) : '')
    }
    setEditing(true)
  }

  const saveEdit = async () => {
    setEditError('')
    setEditSaving(true)
    try {
      await detail.updateVariant(variant.id, {
        name: editName,
        position: variant.position,
        featured_media_id: variant.featured_media_id,
      })
      if (sku) {
        await detail.updateSKU(sku.id, {
          sku_code: editSkuCode,
          barcode: sku.barcode ?? undefined,
          price_minor_units: parsePriceToMinorUnits(editPrice),
          sale_price_minor_units: parseOptionalPriceToMinorUnits(editSalePrice),
          sale_end_date: editSaleEndDate || null,
          currency_code: editCurrency,
          weight_grams: editIsVirtual ? undefined : parseIntOrUndefined(editWeightGrams),
          length_mm: editIsVirtual ? undefined : parseIntOrUndefined(editLengthMM),
          width_mm: editIsVirtual ? undefined : parseIntOrUndefined(editWidthMM),
          height_mm: editIsVirtual ? undefined : parseIntOrUndefined(editHeightMM),
          is_virtual: editIsVirtual,
          tracks_expiry: sku.tracks_expiry,
          case_of_sku_id: sku.case_of_sku_id,
          units_per_case: sku.units_per_case,
          unit_type: sku.unit_type,
          portioned_from_sku_id: sku.portioned_from_sku_id,
          scale_code: sku.scale_code,
          pricing_mode: sku.pricing_mode,
          price_per_kilo_minor_units: sku.price_per_kilo_minor_units,
          status: editStatus,
        })
      }
      setEditing(false)
    } catch (err) {
      setEditError(err instanceof Error ? err.message : 'Failed to save changes')
    } finally {
      setEditSaving(false)
    }
  }

  const selectedImage = productMedia.find(m => m.id === variant.featured_media_id) ?? null
  const [pickerOpen, setPickerOpen] = useState(false)

  const handleSelectImage = async (mediaId: string | null) => {
    setImageError('')
    try {
      await detail.updateVariant(variant.id, {
        name: variant.name,
        position: variant.position,
        featured_media_id: mediaId,
      })
      setPickerOpen(false)
    } catch (err) {
      setImageError(err instanceof Error ? err.message : 'Failed to update image')
    }
  }

  const thumbnailContent = selectedImage ? (
    <img src={selectedImage.url} alt={selectedImage.alt_text} className="w-full h-full object-cover" />
  ) : (
    <Package size={14} className="text-ink-4" />
  )

  return (
    <div className="p-3 rounded-lg border border-line">
      <div className="flex items-center gap-3">
        {productMedia.length > 0 ? (
          <button
            type="button"
            aria-label={`${variant.name} image`}
            onClick={() => setPickerOpen(v => !v)}
            className="w-10 h-10 rounded-lg dark:bg-night-800 bg-gray-100 flex items-center justify-center shrink-0 overflow-hidden"
          >
            {thumbnailContent}
          </button>
        ) : (
          <div className="w-10 h-10 rounded-lg dark:bg-night-800 bg-gray-100 flex items-center justify-center shrink-0 overflow-hidden">
            {thumbnailContent}
          </div>
        )}
        <div className="flex-1 min-w-0">
          <p className="text-xs font-medium text-ink truncate">{variant.name}</p>
          <p className="text-2xs font-mono text-ink-4 truncate">
            {sku ? sku.sku_code : 'No SKU yet'}
          </p>
        </div>
        {sku && sku.sale_price_minor_units != null ? (
          <span className="flex items-center gap-1.5 shrink-0">
            <span className="text-2xs text-ink-4 line-through">
              {formatPrice(sku.price_minor_units, sku.currency_code)}
            </span>
            <span className="text-xs font-medium text-success">
              {formatPrice(sku.sale_price_minor_units, sku.currency_code)}
            </span>
            {sku.sale_end_date && (
              <span className="text-2xs text-ink-4">until {sku.sale_end_date}</span>
            )}
          </span>
        ) : sku && (
          <span className="text-xs font-medium text-ink shrink-0">
            {formatPrice(sku.price_minor_units, sku.currency_code)}
          </span>
        )}
        {sku && (
          <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium capitalize shrink-0', SKU_STATUS_STYLES[sku.status])}>
            {sku.status}
          </span>
        )}
        {sku && <StockBadge skuId={sku.id} />}
        <div className="flex items-center gap-0.5 shrink-0">
          <button aria-label="Edit variant"
            type="button" onClick={openEdit} title="Edit variant"
            className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
          >
            <Edit3 size={12} className="text-ink-3" />
          </button>
          <button aria-label="Remove variant"
            type="button"
            onClick={async () => {
              if (await confirm({ title: `Remove variant "${variant.name}"?`, message: 'Its SKU and stock records are removed too.', confirmLabel: 'Remove variant', destructive: true })) {
                detail.removeVariant(variant.id)
              }
            }}
            title="Remove variant"
            className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
          >
            <Trash2 size={12} className="text-ink-3 hover:text-danger" />
          </button>
        </div>
      </div>

      {editing && (
        // Plain div, not <form> -- same nested-<form>-inside-the-page's-
        // outer-form avoidance as the Add Variant/Add SKU panels.
        <div className="mt-2 space-y-3 p-3 rounded-lg dark:bg-night-800/60 bg-gray-50 border border-line">
          <div>
            <label className={fieldLabelClass}>Variant name</label>
            <input
              type="text" required placeholder="Variant name" value={editName} onChange={e => setEditName(e.target.value)}
              className={fieldInputClass}
            />
          </div>
          {sku && (
            <>
              <div className="space-y-1.5">
                <p className="text-2xs font-semibold text-ink-3 uppercase tracking-wide">Pricing</p>
                <div className="grid grid-cols-2 gap-1.5">
                  <div>
                    <label className={fieldLabelClass}>SKU code</label>
                    <input type="text" required placeholder="SKU code" value={editSkuCode} onChange={e => setEditSkuCode(e.target.value)} className={fieldInputClass} />
                  </div>
                  <div>
                    <label className={fieldLabelClass}>Currency</label>
                    <select value={editCurrency} onChange={e => setEditCurrency(e.target.value)} className={fieldInputClass}>
                      {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                    </select>
                  </div>
                  <div>
                    <label className={fieldLabelClass}>Price</label>
                    <input type="number" step="0.01" min="0" required placeholder="0.00" value={editPrice} onChange={e => setEditPrice(e.target.value)} className={fieldInputClass} />
                  </div>
                  <div>
                    <label className={fieldLabelClass}>Sale price</label>
                    <input type="number" step="0.01" min="0" placeholder="Optional" value={editSalePrice} onChange={e => setEditSalePrice(e.target.value)} className={fieldInputClass} />
                  </div>
                  <div>
                    <label className={fieldLabelClass}>Sale ends</label>
                    <input type="date" title="Sale ends (optional)" value={editSaleEndDate} onChange={e => setEditSaleEndDate(e.target.value)} className={fieldInputClass} />
                  </div>
                  <div>
                    <label className={fieldLabelClass}>Status</label>
                    <select value={editStatus} onChange={e => setEditStatus(e.target.value as SKUStatus)} className={fieldInputClass}>
                      <option value="active">Active</option>
                      <option value="inactive">Inactive</option>
                    </select>
                  </div>
                </div>
              </div>
              {sku && <SKUCurrencyPricesEditor skuId={sku.id} />}
              <div className="space-y-1.5">
                <p className="text-2xs font-semibold text-ink-3 uppercase tracking-wide">Shipping</p>
                <label className="flex items-center gap-1.5 text-2xs text-ink-3">
                  <input type="checkbox" checked={editIsVirtual} onChange={e => setEditIsVirtual(e.target.checked)} />
                  Virtual (no shipping)
                </label>
                {!editIsVirtual && (
                  <div className="grid grid-cols-2 gap-1.5">
                    <div>
                      <label className={fieldLabelClass}>Weight (g)</label>
                      <input type="number" step="1" min="0" placeholder="0" value={editWeightGrams} onChange={e => setEditWeightGrams(e.target.value)} className={fieldInputClass} />
                    </div>
                    <div>
                      <label className={fieldLabelClass}>Length (mm)</label>
                      <input type="number" step="1" min="0" placeholder="0" value={editLengthMM} onChange={e => setEditLengthMM(e.target.value)} className={fieldInputClass} />
                    </div>
                    <div>
                      <label className={fieldLabelClass}>Width (mm)</label>
                      <input type="number" step="1" min="0" placeholder="0" value={editWidthMM} onChange={e => setEditWidthMM(e.target.value)} className={fieldInputClass} />
                    </div>
                    <div>
                      <label className={fieldLabelClass}>Height (mm)</label>
                      <input type="number" step="1" min="0" placeholder="0" value={editHeightMM} onChange={e => setEditHeightMM(e.target.value)} className={fieldInputClass} />
                    </div>
                  </div>
                )}
              </div>
            </>
          )}
          {editError && <p className="text-2xs text-danger">{editError}</p>}
          <div className="flex items-center gap-2 pt-1 border-t border-line">
            <button
              type="button" disabled={editSaving || !editName || (!!sku && !editSkuCode)} onClick={saveEdit}
              className="px-2.5 py-1 rounded bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
            >
              {editSaving ? 'Saving…' : 'Save'}
            </button>
            <button
              type="button" disabled={editSaving} onClick={() => setEditing(false)}
              className="px-2.5 py-1 rounded bg-fill text-2xs text-ink-3 hover:bg-hover transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {pickerOpen && (
        <div className="flex flex-wrap gap-1.5 mt-2 p-2 rounded-lg dark:bg-night-800/60 bg-gray-50 border border-line">
          <button
            type="button"
            onClick={() => handleSelectImage(null)}
            aria-label="No image"
            className={clsx(
              'w-8 h-8 rounded-md border flex items-center justify-center text-2xs text-ink-4 shrink-0',
              !variant.featured_media_id ? 'border-accent' : 'border-line',
            )}
          >
            <X size={12} />
          </button>
          {productMedia.map(m => (
            <button aria-pressed={m.id === variant.featured_media_id}
              key={m.id}
              type="button"
              onClick={() => handleSelectImage(m.id)}
              aria-label={m.alt_text || 'Product image'}
              className={clsx(
                'w-8 h-8 rounded-md overflow-hidden border shrink-0',
                m.id === variant.featured_media_id ? 'border-accent' : 'border-line',
              )}
            >
              <img src={m.url} alt={m.alt_text} className="w-full h-full object-cover" />
            </button>
          ))}
        </div>
      )}

      {imageError && <p className="text-2xs text-danger mt-1.5">{imageError}</p>}

      {sku && (
        <div className="mt-1.5 space-y-1">
          <div className="flex items-center gap-1.5">
            <label htmlFor={`variant-sku-barcode-${sku.id}`} className="text-2xs text-ink-3 shrink-0">
              EAN
            </label>
            <input
              id={`variant-sku-barcode-${sku.id}`} type="text"
              placeholder={sku.scale_code ? 'Not used — see scale code below' : 'Barcode / EAN'}
              disabled={!!sku.scale_code} value={sku.scale_code ? '' : barcodeEdit} onChange={e => setBarcodeEdit(e.target.value)}
              className="flex-1 min-w-0 px-2 py-1 text-2xs rounded bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
            />
            {!sku.scale_code && barcodeEdit !== (sku.barcode ?? '') && (
              <button
                type="button" disabled={barcodeSaving} onClick={saveBarcode}
                className="text-2xs text-accent hover:text-accent/80 transition-colors disabled:opacity-50 shrink-0"
              >
                {barcodeSaving ? 'Saving…' : 'Save'}
              </button>
            )}
          </div>
          {barcodeError && <p className="text-2xs text-danger">{barcodeError}</p>}
          {sku.scale_code ? <ScaleCodeAliasesEditor skuId={sku.id} /> : <BarcodeAliasesEditor skuId={sku.id} />}
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
                Portioned from <span className="font-medium">{sku.portioned_from_sku_id}</span>
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
            <button
              type="button"
              onClick={async () => {
                if (await confirm({ title: `Remove SKU "${sku.sku_code}"?`, message: 'Its stock records are removed too.', confirmLabel: 'Remove SKU', destructive: true })) {
                  detail.removeSKU(sku.id)
                }
              }}
              className="text-2xs text-ink-4 hover:text-danger transition-colors"
            >
              Remove SKU
            </button>
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
              token={token ?? ''} bulkSkuId={sku.id}
              pricePerKiloMinorUnits={sku.price_per_kilo_minor_units} currencyCode={sku.currency_code}
              onCreated={() => detail.refresh()} onClose={() => setBulkAddOpen(false)}
            />
          )}
          {bulkError && <p className="text-2xs text-danger">{bulkError}</p>}
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
              taxRateBasisPoints={taxRateForSku?.rate_basis_points ?? null}
              pricesIncludeTax={tenantSettings?.prices_include_tax ?? false}
            />
          )}
        </div>
      )}

      {!sku && !addingSKU && (
        <button type="button" onClick={() => setAddingSKU(true)} className="text-2xs text-accent hover:text-accent/80 transition-colors mt-2">
          + Add SKU
        </button>
      )}

      {addingSKU && (
        // Plain div, not <form> — same nested-<form>-inside-the-page's-outer-
        // form bug as the Add Variant form above (see comment there).
        <div className="mt-2 space-y-1.5 p-2 rounded-lg dark:bg-night-800/60 bg-gray-50 border border-line">
          <div className="grid grid-cols-2 gap-1.5">
            <div>
              <label className={fieldLabelClass}>SKU code</label>
              <input type="text" required placeholder="SKU code" value={skuCode} onChange={e => setSkuCode(e.target.value)} className={fieldInputClass} />
            </div>
            <div>
              <label className={fieldLabelClass}>Barcode</label>
              <input type="text" placeholder="Optional" value={barcode} onChange={e => setBarcode(e.target.value)} className={fieldInputClass} />
            </div>
            <div>
              <label className={fieldLabelClass}>Price</label>
              <input type="number" step="0.01" min="0" required placeholder="0.00" value={price} onChange={e => setPrice(e.target.value)} className={fieldInputClass} />
            </div>
            <div>
              <label className={fieldLabelClass}>Sale price</label>
              <input type="number" step="0.01" min="0" placeholder="Optional" value={salePrice} onChange={e => setSalePrice(e.target.value)} className={fieldInputClass} />
            </div>
            <div>
              <label className={fieldLabelClass}>Sale ends</label>
              <input type="date" title="Sale ends (optional)" value={saleEndDate} onChange={e => setSaleEndDate(e.target.value)} className={fieldInputClass} />
            </div>
            <div>
              <label className={fieldLabelClass}>Currency</label>
              <select value={currency} onChange={e => setCurrency(e.target.value)} className={fieldInputClass}>
                {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
              </select>
            </div>
          </div>
          <div>
            <label className={fieldLabelClass}>Status</label>
            <select value={status} onChange={e => setStatus(e.target.value as SKUStatus)} className={fieldInputClass}>
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
            </select>
          </div>
          <label className="flex items-center gap-1.5 text-2xs text-ink-3">
            <input type="checkbox" checked={isVirtual} onChange={e => setIsVirtual(e.target.checked)} />
            Virtual (no shipping)
          </label>
          {!isVirtual && (
            <div className="grid grid-cols-2 gap-1.5">
              <div>
                <label className={fieldLabelClass}>Weight (g)</label>
                <input type="number" step="1" min="0" placeholder="0" value={weightGrams} onChange={e => setWeightGrams(e.target.value)} className={fieldInputClass} />
              </div>
              <div>
                <label className={fieldLabelClass}>Length (mm)</label>
                <input type="number" step="1" min="0" placeholder="0" value={lengthMM} onChange={e => setLengthMM(e.target.value)} className={fieldInputClass} />
              </div>
              <div>
                <label className={fieldLabelClass}>Width (mm)</label>
                <input type="number" step="1" min="0" placeholder="0" value={widthMM} onChange={e => setWidthMM(e.target.value)} className={fieldInputClass} />
              </div>
              <div>
                <label className={fieldLabelClass}>Height (mm)</label>
                <input type="number" step="1" min="0" placeholder="0" value={heightMM} onChange={e => setHeightMM(e.target.value)} className={fieldInputClass} />
              </div>
            </div>
          )}
          <label className="flex items-center gap-1.5 text-2xs text-ink-3">
            Item type
            <select
              value={unitType}
              onChange={e => {
                const next = e.target.value as SKUUnitType
                setUnitType(next)
                if (next === 'kilo') { setPortionedFromSKUID(null); setPortionedFromLabel('') }
              }}
              className="px-2 py-1 text-2xs rounded bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
            >
              <option value="unit">Unit</option>
              <option value="box">Box</option>
              <option value="kilo">Kilo (weight pool)</option>
            </select>
          </label>
          {unitType !== 'kilo' && (
            portionedFromSKUID ? (
              <div className="flex items-center gap-1.5 text-2xs text-ink-3">
                <span className="flex-1">Portioned from <span className="font-medium">{portionedFromLabel || portionedFromSKUID}</span></span>
                <button type="button" onClick={() => { setPortionedFromSKUID(null); setPortionedFromLabel('') }} className="hover:underline">Unlink</button>
              </div>
            ) : portionPickerOpen ? (
              <CaseOfSKUPicker
                excludeSKUID=""
                filterSku={s => s.unit_type === 'kilo'}
                onSelect={(s, label) => { setPortionedFromSKUID(s.id); setPortionedFromLabel(label); setPortionPickerOpen(false) }}
                onClose={() => setPortionPickerOpen(false)}
              />
            ) : (
              <button type="button" onClick={() => setPortionPickerOpen(true)} className="text-2xs text-accent hover:text-accent/80 transition-colors">
                + Link to a weight pool
              </button>
            )
          )}
          <div>
            <label className={fieldLabelClass}>Scale code (weight-embedded barcode)</label>
            <input type="text" placeholder="Not sold via a scale barcode" value={scaleCode} onChange={e => setScaleCode(normalizeScaleCodeInput(e.target.value))} maxLength={7} inputMode="numeric" className={fieldInputClass} />
          </div>
          <div className="grid grid-cols-2 gap-1.5">
            <label className="flex flex-col gap-0.5 text-2xs text-ink-3">
              Pricing
              <select value={pricingMode} onChange={e => setPricingMode(e.target.value as SKUPricingMode)} className={fieldInputClass}>
                <option value="fixed">Fixed / package</option>
                <option value="per_weight">Per kilo</option>
              </select>
            </label>
            {pricingMode === 'per_weight' && (
              <div>
                <label className={fieldLabelClass}>Price/kilo ({currency})</label>
                <input type="number" step="0.01" min="0" required value={pricePerKilo} onChange={e => setPricePerKilo(e.target.value)} className={fieldInputClass} />
              </div>
            )}
          </div>
          {pricingMode === 'per_weight' && !scaleCode && (
            <p className="text-2xs text-warning">Set a scale code above too — per-kilo pricing only applies once a package's weight can actually be scanned.</p>
          )}
          {error && <p className="text-2xs text-danger">{error}</p>}
          <div className="flex gap-1.5">
            <button type="button" onClick={handleAddSKU} disabled={submitting} className="flex-1 py-1 rounded bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors">
              {submitting ? 'Adding…' : 'Add'}
            </button>
            <button type="button" onClick={() => setAddingSKU(false)} className="px-2 py-1 rounded bg-fill text-2xs text-ink-3 transition-colors">
              Cancel
            </button>
          </div>
        </div>
      )}
      {confirmDialog}
    </div>
  )
}
