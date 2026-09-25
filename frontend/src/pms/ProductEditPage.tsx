import { useState, useEffect } from 'react'
import { useQueryEnum } from '../routing/useQueryState'
import { Search, ArrowLeft, Plug, X, RefreshCw } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { sortCategoriesHierarchically } from './categoryTree'
import { useProductDetail } from './hooks/useProductDetail'
import { ProductBolTab } from './ProductBolTab'
import * as pmsApi from './api'
import * as integrationsApi from '../integrations/api'
import type { Connection } from '../integrations/types'
import type { Category, Product, ProductStatus, SKUStatus, SKUUnitType, SKUPricingMode, VariantWithSKUs } from './types'
import { useTaxClasses } from '../tax/hooks/useTaxClasses'
import { useTaxRates } from '../tax/hooks/useTaxRates'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import { CURRENCIES, formatMinorUnitsAsInput, normalizeScaleCodeInput, parseIntOrUndefined, parseOptionalPriceToMinorUnits, parsePriceToMinorUnits } from './productFormat'
import { ProductWooCommerceTab } from './ProductWooCommerceTab'
import { FormCard, ToggleRow } from './FormCard'
import { MediaSection } from './MediaSection'
import { CategoryMarginEditor, MarginInsightCard, PriceHistoryCard, ProductHistoryCard } from './ProductPricingCards'
import { CaseOfSKUPicker, VariantsSection } from './VariantsSection'
import { BarcodeAliasesEditor, ScaleCodeAliasesEditor } from './SkuAliasEditors'

// ── Product Edit Page ────────────────────────────────────────────────────────
//
// Dedicated full page (not a slide-over) for creating/editing a product.
// "General" tab holds the shared data every provider needs (name, price,
// variants...). Additional tabs hold provider-specific values — only
// WooCommerce for now, one tab per connected integration eventually
// (Shopify, Lightspeed, ...), each mapping the same product to that
// channel's own fields without polluting the generic model.

const EDITOR_TABS = ['general', 'woocommerce', 'bol'] as const

type EditorTab = typeof EDITOR_TABS[number]

interface ProductEditorProps {
  productId: string | null
  categories: Category[]
  onCreateCategory: (payload: { name: string }) => Promise<{ id: string; name: string }>
  onUpdateCategory: (id: string, payload: { name: string; parent_id?: string | null; target_markup_basis_points?: number | null }) => Promise<Category>
  onCreate: (payload: { name: string; description?: string; category_id?: string | null; status: ProductStatus; has_variants?: boolean; sync_variants_as_woo_variations?: boolean; tags?: string[] }) => Promise<Product>
  onUpdate: (id: string, payload: { name: string; slug?: string; description?: string; category_id?: string | null; status: ProductStatus; has_variants?: boolean; sync_variants_as_woo_variations?: boolean; tags?: string[] }) => Promise<Product>
  onClose: () => void
}

export function ProductEditPage({ productId, categories, onCreateCategory, onUpdateCategory, onCreate, onUpdate, onClose }: ProductEditorProps) {
  const { token } = useAuth()
  const [savedProductId, setSavedProductId] = useState<string | null>(productId)
  const detail = useProductDetail(token ?? '', savedProductId)
  const { classes: taxClasses } = useTaxClasses(token ?? '')
  const { rates: taxRates } = useTaxRates(token ?? '')
  const { settings: tenantSettings } = useTenantSettings(token ?? '')

  const existing = detail.detail?.product
  const [name, setName] = useState(existing?.name ?? '')
  // Only ever sent to the server on an existing product's save (create
  // always derives slug from name server-side) -- this is what lets a
  // duplicated product's stuck "-copy" slug be fixed after the fact.
  const [slug, setSlug] = useState(existing?.slug ?? '')
  const [description, setDescription] = useState(existing?.description ?? '')
  const [categoryId, setCategoryId] = useState<string>(existing?.category_id ?? '')
  const [status, setStatus] = useState<ProductStatus>(existing?.status ?? 'draft')
  const [tags, setTags] = useState<string[]>(existing?.tags ?? [])
  const [tagInput, setTagInput] = useState('')
  const [newCategoryName, setNewCategoryName] = useState('')
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')
  const [requestedTab, setActiveTab] = useQueryEnum<EditorTab>('tab', EDITOR_TABS, 'general')
  const [resyncing, setResyncing] = useState(false)
  const [resyncMessage, setResyncMessage] = useState('')

  // Integration tabs only ever appear for a provider that's actually
  // connected — an unconnected/disabled integration gets no tab at all,
  // rather than a tab that just tells you to go connect it.
  const [wooConnection, setWooConnection] = useState<Connection | null>(null)
  const [bolConnection, setBolConnection] = useState<Connection | null>(null)
  // ?tab= can name a channel tab this product doesn't have (stale link, or
  // connections still loading) -- show General rather than an empty page.
  const activeTab: EditorTab =
    (requestedTab === 'woocommerce' && !(savedProductId && wooConnection)) ||
    (requestedTab === 'bol' && !(savedProductId && bolConnection))
      ? 'general'
      : requestedTab
  useEffect(() => {
    if (!token) return
    integrationsApi.listConnections(token)
      .then(conns => {
        setWooConnection(conns.find(c => c.provider === 'woocommerce' && c.status === 'connected') ?? null)
        setBolConnection(conns.find(c => c.provider === 'bol' && c.status === 'connected') ?? null)
      })
      .catch(() => { setWooConnection(null); setBolConnection(null) })
  }, [token])

  // Whether to show the full named-variants manager vs. the flat single-SKU
  // form. For a brand-new product this is a user choice made before saving;
  // for an existing product it's the server's own has_variants field —
  // source of truth lives in the DB, not inferred from variant count.
  const [variantsMode, setVariantsMode] = useState(false)
  // Only matters once variantsMode is true and a WooCommerce store is
  // connected -- see Product.sync_variants_as_woo_variations for the full
  // shape this controls.
  const [syncVariantsAsWooVariations, setSyncVariantsAsWooVariations] = useState(false)

  // Simple-mode SKU fields (only relevant when variantsMode is false).
  const [skuCode, setSkuCode] = useState('')
  const [barcode, setBarcode] = useState('')
  const [price, setPrice] = useState('')
  const [salePrice, setSalePrice] = useState('')
  const [saleEndDate, setSaleEndDate] = useState('')
  const [currency, setCurrency] = useState('USD')
  const [skuStatus, setSkuStatus] = useState<SKUStatus>('active')
  const [isVirtual, setIsVirtual] = useState(false)
  const [weightGrams, setWeightGrams] = useState('')
  const [lengthMM, setLengthMM] = useState('')
  const [widthMM, setWidthMM] = useState('')
  const [heightMM, setHeightMM] = useState('')
  // Case-pack relationship (see pms.SKU.CaseOfSKUID) -- only editable once
  // the SKU already exists, since it links to another already-existing SKU.
  const [caseOfSKUID, setCaseOfSKUID] = useState<string | null>(null)
  const [caseOfSKULabel, setCaseOfSKULabel] = useState('')
  const [unitsPerCase, setUnitsPerCase] = useState('')
  const [casePickerOpen, setCasePickerOpen] = useState(false)
  // Weight pool (see pms.SKU.UnitType/PortionedFromSKUID) -- a SKU is
  // either a weight pool itself (unit_type kilo), or portioned from one,
  // never both.
  const [unitType, setUnitType] = useState<SKUUnitType>('unit')
  const [portionedFromSKUID, setPortionedFromSKUID] = useState<string | null>(null)
  const [portionedFromLabel, setPortionedFromLabel] = useState('')
  const [portionPickerOpen, setPortionPickerOpen] = useState(false)
  // Weight-embedded (butcher/deli scale) barcode support (see
  // pms.SKU.ScaleCode/PricingMode/PricePerKiloMinorUnits) -- scaleCode '' means
  // this SKU isn't sold that way; pricingMode/pricePerKilo only matter when set.
  const [scaleCode, setScaleCode] = useState('')
  const [pricingMode, setPricingMode] = useState<SKUPricingMode>('fixed')
  const [pricePerKilo, setPricePerKilo] = useState('')
  // null means "use the tenant's default tax class" (see TaxClass.is_default)
  // -- '' in the <select> represents that same choice.
  const [taxClassId, setTaxClassId] = useState<string | null>(null)

  // Sync local form state once the product detail loads (edit mode) — runs once.
  const [hydrated, setHydrated] = useState(false)
  if (!hydrated && existing) {
    setName(existing.name)
    setSlug(existing.slug)
    setDescription(existing.description)
    setCategoryId(existing.category_id ?? '')
    setStatus(existing.status)
    setTags(existing.tags ?? [])
    setVariantsMode(existing.has_variants)
    setSyncVariantsAsWooVariations(existing.sync_variants_as_woo_variations ?? false)
    const variants = detail.detail?.variants ?? []
    if (variants.length === 1) {
      const sku = variants[0].skus?.[0]
      if (sku) {
        setSkuCode(sku.sku_code)
        setBarcode(sku.barcode ?? '')
        setPrice(formatMinorUnitsAsInput(sku.price_minor_units))
        setSalePrice(sku.sale_price_minor_units != null ? formatMinorUnitsAsInput(sku.sale_price_minor_units) : '')
        setSaleEndDate(sku.sale_end_date ?? '')
        setCurrency(sku.currency_code)
        setSkuStatus(sku.status)
        setIsVirtual(sku.is_virtual)
        setWeightGrams(sku.weight_grams != null ? String(sku.weight_grams) : '')
        setLengthMM(sku.length_mm != null ? String(sku.length_mm) : '')
        setWidthMM(sku.width_mm != null ? String(sku.width_mm) : '')
        setHeightMM(sku.height_mm != null ? String(sku.height_mm) : '')
        setCaseOfSKUID(sku.case_of_sku_id ?? null)
        setUnitsPerCase(sku.units_per_case != null ? String(sku.units_per_case) : '')
        setUnitType(sku.unit_type ?? 'unit')
        setPortionedFromSKUID(sku.portioned_from_sku_id ?? null)
        setScaleCode(sku.scale_code ?? '')
        setPricingMode(sku.pricing_mode ?? 'fixed')
        setPricePerKilo(sku.price_per_kilo_minor_units != null ? formatMinorUnitsAsInput(sku.price_per_kilo_minor_units) : '')
        setTaxClassId(sku.tax_class_id ?? null)
      }
    }
    setHydrated(true)
  }

  // A brand-new product (productId===null, so nothing to hydrate from
  // above) still needs its currency seeded from the tenant's own default
  // rather than sitting on the initial useState('USD') forever -- gated on
  // tenantSettings actually having loaded (not just !loading, which is
  // still false for an instant before the fetch effect kicks in) so this
  // doesn't fire-and-mark-done before the real value arrives.
  const [currencyDefaulted, setCurrencyDefaulted] = useState(false)
  if (!currencyDefaulted && productId === null && tenantSettings) {
    if (tenantSettings.default_currency) setCurrency(tenantSettings.default_currency)
    setCurrencyDefaulted(true)
  }

  const singleVariant: VariantWithSKUs | null =
    !variantsMode && detail.detail?.variants.length === 1 ? detail.detail.variants[0] : null
  const currentSku = singleVariant?.skus?.[0] ?? null

  const defaultTaxClass = taxClasses.find(c => c.is_default) ?? null
  const effectiveTaxClassId = taxClassId ?? defaultTaxClass?.id ?? null
  const taxRateForPreview = effectiveTaxClassId && tenantSettings?.default_tax_country
    ? taxRates.find(r => r.tax_class_id === effectiveTaxClassId && r.country === tenantSettings.default_tax_country)
    : undefined
  const priceMinorUnitsForPreview = price.trim() === '' ? null : parsePriceToMinorUnits(price)
  const taxPreview = taxRateForPreview && priceMinorUnitsForPreview != null
    ? (() => {
        const rate = taxRateForPreview.rate_basis_points / 10000
        const inclusiveMinor = tenantSettings?.prices_include_tax
          ? priceMinorUnitsForPreview
          : Math.round(priceMinorUnitsForPreview * (1 + rate))
        const exclusiveMinor = tenantSettings?.prices_include_tax
          ? Math.round(priceMinorUnitsForPreview / (1 + rate))
          : priceMinorUnitsForPreview
        return {
          exclusive: exclusiveMinor / 100,
          inclusive: inclusiveMinor / 100,
          ratePercent: (taxRateForPreview.rate_basis_points / 100).toFixed(2),
          country: taxRateForPreview.country,
        }
      })()
    : null

  const addTag = (raw: string) => {
    const value = raw.trim()
    if (value && !tags.includes(value)) setTags(prev => [...prev, value])
    setTagInput('')
  }
  const removeTag = (value: string) => setTags(prev => prev.filter(t => t !== value))
  const handleTagInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      addTag(tagInput)
    } else if (e.key === 'Backspace' && !tagInput && tags.length > 0) {
      removeTag(tags[tags.length - 1])
    }
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      const payload = {
        name, description, category_id: categoryId || null, status, has_variants: variantsMode,
        sync_variants_as_woo_variations: syncVariantsAsWooVariations, tags,
        // Only meaningful on update -- create always derives slug from
        // name server-side, and only sending it when it actually changed
        // from what loaded avoids a spurious uniqueness check against the
        // product's own current slug.
        ...(savedProductId && slug !== existing?.slug ? { slug } : {}),
      }
      // A virtual SKU never ships, so it can never carry weight/dimensions
      // (enforced server-side too — see ErrVirtualSKUCannotHaveDimensions).
      const dimensions = isVirtual
        ? { weight_grams: undefined, length_mm: undefined, width_mm: undefined, height_mm: undefined }
        : {
            weight_grams: parseIntOrUndefined(weightGrams),
            length_mm: parseIntOrUndefined(lengthMM),
            width_mm: parseIntOrUndefined(widthMM),
            height_mm: parseIntOrUndefined(heightMM),
          }

      if (!savedProductId) {
        const product = await onCreate(payload)
        if (!variantsMode) {
          const variant = await pmsApi.createVariant(token ?? '', product.id, { name: product.name })
          if (skuCode) {
            await pmsApi.createSKU(token ?? '', variant.id, {
              sku_code: skuCode,
              barcode: barcode || undefined,
              price_minor_units: parsePriceToMinorUnits(price),
              sale_price_minor_units: parseOptionalPriceToMinorUnits(salePrice),
              sale_end_date: saleEndDate || null,
              currency_code: currency,
              is_virtual: isVirtual,
              ...dimensions,
              unit_type: unitType,
              portioned_from_sku_id: portionedFromSKUID,
              scale_code: scaleCode || null,
              pricing_mode: pricingMode,
              price_per_kilo_minor_units: pricingMode === 'per_weight' ? parseOptionalPriceToMinorUnits(pricePerKilo) : null,
              tax_class_id: taxClassId,
              status: skuStatus,
            })
          }
        }
        // Set this last, after the variant/SKU writes above complete --
        // useProductDetail's auto-fetch fires the instant `savedProductId`
        // changes (its `productId` prop), so setting it any earlier races
        // that fetch against the still-in-flight creates above and can
        // land on an empty `variants: []` permanently (no later refetch
        // ever corrects it, since nothing besides this transition
        // triggers one).
        setSavedProductId(product.id)
        return
      }

      const updatedProduct = await onUpdate(savedProductId, payload)
      // The `hydrated` sync-from-`existing` block above only ever runs
      // once (by design, so it doesn't clobber in-progress edits every
      // time `existing` changes) -- so a server-side transform like slug
      // regeneration would otherwise sit in `detail` unseen until a full
      // page reload. Patch every field the save just touched directly
      // from the response, which -- unlike a subsequent GET -- already
      // reflects transforms UpdateProductUseCase applied in memory before
      // persisting (e.g. slug regeneration/disambiguation).
      setName(updatedProduct.name)
      setSlug(updatedProduct.slug)
      setDescription(updatedProduct.description)
      setCategoryId(updatedProduct.category_id ?? '')
      setStatus(updatedProduct.status)
      setTags(updatedProduct.tags ?? [])
      // addSKU/updateSKU below already refresh `detail` themselves, but a
      // save with no SKU changes wouldn't otherwise touch it -- without
      // this, `existing` (used for the slug-changed diff above) stays
      // stale for a second save within the same edit session.
      detail.refresh()
      if (!variantsMode) {
        if (singleVariant) {
          const currentSku = singleVariant.skus?.[0]
          if (currentSku) {
            await detail.updateSKU(currentSku.id, {
              sku_code: skuCode,
              barcode: barcode || undefined,
              price_minor_units: parsePriceToMinorUnits(price),
              sale_price_minor_units: parseOptionalPriceToMinorUnits(salePrice),
              sale_end_date: saleEndDate || null,
              currency_code: currency,
              is_virtual: isVirtual,
              tracks_expiry: currentSku.tracks_expiry,
              ...dimensions,
              case_of_sku_id: caseOfSKUID,
              units_per_case: caseOfSKUID ? parseIntOrUndefined(unitsPerCase) ?? null : null,
              unit_type: unitType,
              portioned_from_sku_id: portionedFromSKUID,
              scale_code: scaleCode || null,
              pricing_mode: pricingMode,
              price_per_kilo_minor_units: pricingMode === 'per_weight' ? parseOptionalPriceToMinorUnits(pricePerKilo) : null,
              tax_class_id: taxClassId,
              status: skuStatus,
            })
          } else if (skuCode) {
            await detail.addSKU(singleVariant.id, {
              sku_code: skuCode,
              barcode: barcode || undefined,
              price_minor_units: parsePriceToMinorUnits(price),
              sale_price_minor_units: parseOptionalPriceToMinorUnits(salePrice),
              sale_end_date: saleEndDate || null,
              currency_code: currency,
              is_virtual: isVirtual,
              ...dimensions,
              scale_code: scaleCode || null,
              pricing_mode: pricingMode,
              price_per_kilo_minor_units: pricingMode === 'per_weight' ? parseOptionalPriceToMinorUnits(pricePerKilo) : null,
              tax_class_id: taxClassId,
              status: skuStatus,
            })
          }
        } else if (skuCode) {
          const variant = await pmsApi.createVariant(token ?? '', savedProductId, { name })
          await pmsApi.createSKU(token ?? '', variant.id, {
            sku_code: skuCode,
            barcode: barcode || undefined,
            price_minor_units: parsePriceToMinorUnits(price),
            sale_price_minor_units: parseOptionalPriceToMinorUnits(salePrice),
            sale_end_date: saleEndDate || null,
            currency_code: currency,
            is_virtual: isVirtual,
            ...dimensions,
            scale_code: scaleCode || null,
            pricing_mode: pricingMode,
            price_per_kilo_minor_units: pricingMode === 'per_weight' ? parseOptionalPriceToMinorUnits(pricePerKilo) : null,
            tax_class_id: taxClassId,
            status: skuStatus,
          })
          detail.refresh()
        }
      }
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save product')
    } finally {
      setSaving(false)
    }
  }

  const [enablingVariants, setEnablingVariants] = useState(false)
  const [enableVariantsError, setEnableVariantsError] = useState('')

  // For an already-saved product, has_variants must be persisted before the
  // variants manager becomes interactive — otherwise the first "+ Add
  // Variant" click hits the server's own invariant (product already has 1
  // variant from flat-SKU mode, has_variants still false there) and fails.
  // For a brand-new, unsaved product there's nothing to persist yet — the
  // flag goes out with the initial create payload on Save.
  const handleEnableVariants = async () => {
    if (!savedProductId) {
      setVariantsMode(true)
      return
    }
    setEnableVariantsError('')
    setEnablingVariants(true)
    try {
      await onUpdate(savedProductId, { name, description, category_id: categoryId || null, status, has_variants: true, tags })
      setVariantsMode(true)
    } catch (err) {
      setEnableVariantsError(err instanceof Error ? err.message : 'Failed to enable variants')
    } finally {
      setEnablingVariants(false)
    }
  }

  const handleAddCategory = async () => {
    if (!newCategoryName.trim()) return
    try {
      const category = await onCreateCategory({ name: newCategoryName.trim() })
      setCategoryId(category.id)
      setNewCategoryName('')
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create category')
    }
  }

  const handleResync = async () => {
    if (!savedProductId) return
    setResyncing(true)
    setResyncMessage('')
    try {
      await integrationsApi.resyncProductToChannels(token ?? '', savedProductId)
      setResyncMessage('Resync started.')
    } catch (err) {
      setResyncMessage(err instanceof Error ? err.message : 'Failed to resync')
    } finally {
      setResyncing(false)
    }
  }

  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'
  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Products
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {savedProductId ? name || 'Edit product' : 'New product'}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          {resyncMessage && <span className="text-2xs text-ink-4 mr-1">{resyncMessage}</span>}
          {savedProductId && wooConnection && (
            <button
              type="button" onClick={handleResync} disabled={resyncing}
              title="Force-push this product's current fields and stock to every linked channel, even if nothing changed"
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
            >
              <RefreshCw size={12} className={resyncing ? 'animate-spin' : undefined} />
              {resyncing ? 'Resyncing…' : 'Resync to channels'}
            </button>
          )}
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
          >
            Discard
          </button>
          <button
            type="submit" form="product-general-form" disabled={saving || activeTab !== 'general'}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : savedProductId ? 'Save' : 'Create product'}
          </button>
        </div>
      </div>

      <div className="flex items-center gap-1 px-5 border-b border-line bg-surface shrink-0">
        <button aria-pressed={activeTab === 'general'}
          onClick={() => setActiveTab('general')}
          className={clsx(
            'px-3 py-2.5 text-xs font-medium border-b-2 -mb-px transition-colors',
            activeTab === 'general'
              ? 'border-accent text-accent'
              : 'border-transparent text-ink-4 hover:text-ink-2',
          )}
        >
          General
        </button>
        {savedProductId && wooConnection && (
          <button aria-pressed={activeTab === 'woocommerce'}
            onClick={() => setActiveTab('woocommerce')}
            className={clsx(
              'flex items-center gap-1.5 px-3 py-2.5 text-xs font-medium border-b-2 -mb-px transition-colors',
              activeTab === 'woocommerce'
                ? 'border-accent text-accent'
                : 'border-transparent text-ink-4 hover:text-ink-2',
            )}
          >
            <Plug size={12} /> WooCommerce
          </button>
        )}
        {savedProductId && bolConnection && (
          <button aria-pressed={activeTab === 'bol'}
            onClick={() => setActiveTab('bol')}
            className={clsx(
              'flex items-center gap-1.5 px-3 py-2.5 text-xs font-medium border-b-2 -mb-px transition-colors',
              activeTab === 'bol'
                ? 'border-accent text-accent'
                : 'border-transparent text-ink-4 hover:text-ink-2',
            )}
          >
            <Plug size={12} /> Bol.com
          </button>
        )}
      </div>

      {formError && (
        // Rendered here (shrink-0, above the scrolling form body) rather
        // than inline in the General tab -- a save error must be visible
        // without the user having to scroll past every FormCard to the
        // bottom of the page to notice the save silently failed.
        <div className="px-6 pt-3 shrink-0">
          <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-3 py-2">{formError}</p>
        </div>
      )}

      <div className="flex-1 overflow-y-auto">
        <div className={clsx('max-w-5xl mx-auto p-6', activeTab !== 'general' && 'hidden')}>
          {renderGeneralTab()}
        </div>
        {activeTab === 'woocommerce' && savedProductId && wooConnection && (
          <div className="max-w-5xl mx-auto p-6">
            <ProductWooCommerceTab
              productId={savedProductId} connection={wooConnection} sku={currentSku}
              hasVariants={variantsMode} variants={detail.detail?.variants ?? []}
            />
          </div>
        )}
        {activeTab === 'bol' && savedProductId && bolConnection && (
          <div className="max-w-5xl mx-auto p-6">
            <ProductBolTab
              productId={savedProductId} connection={bolConnection} sku={currentSku} productName={existing?.name}
              onSaveBarcode={async (skuId, newBarcode) => {
                if (!currentSku) return
                await detail.updateSKU(skuId, {
                  sku_code: currentSku.sku_code,
                  barcode: newBarcode || undefined,
                  price_minor_units: currentSku.price_minor_units,
                  sale_price_minor_units: currentSku.sale_price_minor_units,
                  sale_end_date: currentSku.sale_end_date,
                  currency_code: currentSku.currency_code,
                  weight_grams: currentSku.weight_grams,
                  length_mm: currentSku.length_mm,
                  width_mm: currentSku.width_mm,
                  height_mm: currentSku.height_mm,
                  is_virtual: currentSku.is_virtual,
                  tracks_expiry: currentSku.tracks_expiry,
                  case_of_sku_id: currentSku.case_of_sku_id,
                  units_per_case: currentSku.units_per_case,
                  unit_type: currentSku.unit_type,
                  portioned_from_sku_id: currentSku.portioned_from_sku_id,
                  scale_code: currentSku.scale_code,
                  pricing_mode: currentSku.pricing_mode,
                  price_per_kilo_minor_units: currentSku.price_per_kilo_minor_units,
                  tax_class_id: currentSku.tax_class_id,
                  status: currentSku.status,
                })
              }}
            />
          </div>
        )}
      </div>
    </div>
  )

  function renderGeneralTab() {
    const showFlatSKU = !variantsMode && (savedProductId ? singleVariant !== null || detail.detail?.variants.length === 0 : true)

    return (
      <form id="product-general-form" onSubmit={handleSave}>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
          {/* Main column */}
          <div className="lg:col-span-2 space-y-5">
            <FormCard title="General">
              <div>
                <label htmlFor="product-name" className={labelClass}>Name</label>
                <input
                  id="product-name"
                  type="text" required value={name} onChange={e => setName(e.target.value)}
                  className={inputClass}
                />
              </div>
              {savedProductId && (
                <div>
                  <label htmlFor="product-slug" className={labelClass}>Slug</label>
                  <input
                    id="product-slug"
                    type="text" value={slug} onChange={e => setSlug(e.target.value)}
                    pattern="^[a-z0-9]+(-[a-z0-9]+)*$"
                    title="Lowercase letters, numbers, and single dashes only"
                    className={inputClass}
                  />
                </div>
              )}
              <div>
                <label htmlFor="product-description" className={labelClass}>Description</label>
                <textarea
                  id="product-description"
                  value={description} onChange={e => setDescription(e.target.value)} rows={3}
                  className={inputClass}
                />
              </div>
              <div>
                <label htmlFor="product-tags" className={labelClass}>Tags</label>
                <div className={clsx(inputClass, 'flex flex-wrap items-center gap-1.5 h-auto min-h-[2.25rem] py-1.5')}>
                  {tags.map(tag => (
                    <span
                      key={tag}
                      className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-2xs bg-hover text-ink-2"
                    >
                      {tag}
                      <button type="button" onClick={() => removeTag(tag)} aria-label={`Remove tag ${tag}`}>
                        <X size={10} className="text-ink-4" />
                      </button>
                    </span>
                  ))}
                  <input
                    id="product-tags"
                    type="text"
                    value={tagInput}
                    onChange={e => setTagInput(e.target.value)}
                    onKeyDown={handleTagInputKeyDown}
                    onBlur={() => addTag(tagInput)}
                    placeholder={tags.length === 0 ? 'summer, sale…' : ''}
                    className="flex-1 min-w-[6rem] bg-transparent outline-none focus-visible:ring-2 focus-visible:ring-focus/60 text-xs text-ink-2"
                  />
                </div>
                <p className="mt-1.5 text-2xs text-ink-4">
                  Press Enter or comma to add. Used by Sales Channels' "by tag" enrollment rule.
                </p>
              </div>
            </FormCard>

            <FormCard title="Media">
              {savedProductId ? (
                <MediaSection productId={savedProductId} />
              ) : (
                <p className="text-2xs text-ink-4">Save the product first, then add images.</p>
              )}
            </FormCard>

            {savedProductId && <ProductHistoryCard productId={savedProductId} />}

            <FormCard title="Variants">
              {variantsMode ? (
                <>
                  {wooConnection && (
                    <label className="flex items-start gap-2 mb-3 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={syncVariantsAsWooVariations}
                        onChange={e => setSyncVariantsAsWooVariations(e.target.checked)}
                        className="mt-0.5"
                      />
                      <span className="text-2xs text-ink-3">
                        <span className="font-medium text-ink-2">Sync variants as WooCommerce variations</span>
                        <br />
                        On: one Woo product with a variation per variant. Off (default): each variant becomes its own
                        separate Woo product. Changing this after the product is already synced deletes the old
                        WooCommerce listing(s) and recreates them in the new shape.
                      </span>
                    </label>
                  )}
                  {savedProductId ? (
                    <VariantsSection detail={detail} productId={savedProductId} />
                  ) : (
                    <p className="text-2xs text-ink-4">Save the product first, then add variants below.</p>
                  )}
                  {!savedProductId && (
                    <button
                      type="button"
                      onClick={() => setVariantsMode(false)}
                      className="text-2xs text-ink-4 hover:text-ink-2 transition-colors"
                    >
                      ← Actually, no variants
                    </button>
                  )}
                </>
              ) : (
                <div>
                  <button
                    type="button"
                    onClick={handleEnableVariants}
                    disabled={enablingVariants}
                    className="text-xs font-medium text-accent hover:text-accent/80 transition-colors disabled:opacity-50"
                  >
                    {enablingVariants ? 'Enabling…' : '+ Add variants like size or color'}
                  </button>
                  {enableVariantsError && <p className="text-2xs text-danger mt-1.5">{enableVariantsError}</p>}
                </div>
              )}
            </FormCard>

            {showFlatSKU && (
              <FormCard title="Pricing & SKU">
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label htmlFor="product-sku-code" className={labelClass}>SKU code</label>
                    <input
                      id="product-sku-code"
                      type="text" required={!variantsMode} placeholder="e.g. SHIRT-BLK-M" value={skuCode} onChange={e => setSkuCode(e.target.value)}
                      className={inputClass}
                    />
                  </div>
                  <div>
                    <label htmlFor="product-barcode" className={labelClass}>Barcode / EAN</label>
                    <input
                      id="product-barcode"
                      type="text" placeholder={scaleCode ? 'Not used — see Weight-embedded barcode below' : 'Optional'}
                      disabled={!!scaleCode} value={scaleCode ? '' : barcode} onChange={e => setBarcode(e.target.value)}
                      className={inputClass}
                    />
                    {scaleCode && (
                      <p className="text-2xs text-ink-4 mt-1">
                        Not used for this SKU — its real printed barcode changes per package (weight baked in).
                      </p>
                    )}
                    {currentSku && !scaleCode && <div className="mt-1"><BarcodeAliasesEditor skuId={currentSku.id} /></div>}
                  </div>
                  <div>
                    <label htmlFor="product-price" className={labelClass}>Price</label>
                    <input
                      id="product-price"
                      type="number" step="0.01" min="0" required={!variantsMode} placeholder="0.00" value={price} onChange={e => setPrice(e.target.value)}
                      className={inputClass}
                    />
                    {taxPreview && (
                      <p className="mt-1 text-2xs text-ink-4">
                        Excl. tax: {currency} {taxPreview.exclusive.toFixed(2)} · Incl. tax: {currency} {taxPreview.inclusive.toFixed(2)} ({taxPreview.ratePercent}% {taxPreview.country})
                      </p>
                    )}
                  </div>
                  <div>
                    <label htmlFor="product-sale-price" className={labelClass}>Sale price (optional)</label>
                    <input
                      id="product-sale-price"
                      type="number" step="0.01" min="0" placeholder="None" value={salePrice} onChange={e => setSalePrice(e.target.value)}
                      className={inputClass}
                    />
                  </div>
                  <div>
                    <label htmlFor="product-sale-end-date" className={labelClass}>Sale ends (optional)</label>
                    <input
                      id="product-sale-end-date"
                      type="date" value={saleEndDate} onChange={e => setSaleEndDate(e.target.value)}
                      className={inputClass}
                    />
                  </div>
                  <div>
                    <label htmlFor="product-currency" className={labelClass}>Currency</label>
                    <select id="product-currency" value={currency} onChange={e => setCurrency(e.target.value)} className={inputClass}>
                      {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                    </select>
                  </div>
                  <div>
                    <label htmlFor="product-tax-class" className={labelClass}>Tax class</label>
                    <select
                      id="product-tax-class"
                      value={taxClassId ?? ''}
                      onChange={e => setTaxClassId(e.target.value || null)}
                      className={inputClass}
                    >
                      <option value="">Tenant default{defaultTaxClass ? ` (${defaultTaxClass.name})` : ''}</option>
                      {taxClasses.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                    </select>
                  </div>
                  <div>
                    <label htmlFor="product-sku-status" className={labelClass}>SKU status</label>
                    <select id="product-sku-status" value={skuStatus} onChange={e => setSkuStatus(e.target.value as SKUStatus)} className={inputClass}>
                      <option value="active">Active</option>
                      <option value="inactive">Inactive</option>
                    </select>
                  </div>
                </div>
              </FormCard>
            )}

            {showFlatSKU && currentSku && (
              <MarginInsightCard
                skuId={currentSku.id} currency={currency}
                taxRateBasisPoints={taxRateForPreview?.rate_basis_points ?? null}
                pricesIncludeTax={tenantSettings?.prices_include_tax ?? false}
              />
            )}

            {showFlatSKU && currentSku && (
              <PriceHistoryCard skuId={currentSku.id} currency={currency} />
            )}

            {showFlatSKU && currentSku && (
              <FormCard
                title="Case pack"
                description="If a supplier sells this as a sealed case/box of another product in your catalog, link it here — lets you split case stock into unit stock later."
              >
                {caseOfSKUID ? (
                  <div className="flex items-center gap-2">
                    <span className="flex-1 text-xs text-ink">
                      Case of <span className="font-medium">{caseOfSKULabel || caseOfSKUID}</span>
                    </span>
                    <div className="w-24">
                      <label htmlFor="product-units-per-case" className={labelClass}>Units/case</label>
                      <input
                        id="product-units-per-case"
                        type="number" step="1" min="1" required value={unitsPerCase} onChange={e => setUnitsPerCase(e.target.value)}
                        className={inputClass}
                      />
                    </div>
                    <button
                      type="button"
                      onClick={() => { setCaseOfSKUID(null); setCaseOfSKULabel(''); setUnitsPerCase('') }}
                      className="text-2xs text-ink-4 hover:underline"
                    >
                      Remove
                    </button>
                  </div>
                ) : casePickerOpen ? (
                  <CaseOfSKUPicker
                    excludeSKUID={currentSku.id}
                    onSelect={(sku, label) => { setCaseOfSKUID(sku.id); setCaseOfSKULabel(label); setCasePickerOpen(false) }}
                    onClose={() => setCasePickerOpen(false)}
                  />
                ) : (
                  <button
                    type="button"
                    onClick={() => setCasePickerOpen(true)}
                    className="flex items-center gap-1 text-xs font-medium text-accent hover:text-accent/80 transition-colors"
                  >
                    <Search size={12} /> This is a case of another product
                  </button>
                )}
              </FormCard>
            )}

            {showFlatSKU && (
              <FormCard
                title="Weight-embedded barcode"
                description="For a butcher/deli scale that prints a barcode with the package's weight baked in — every package gets a different full barcode, so scanning it is matched by this scale code instead."
              >
                <div>
                  <label htmlFor="product-scale-code" className={labelClass}>Scale code</label>
                  <input
                    id="product-scale-code"
                    type="text" placeholder="Not sold via a scale barcode" value={scaleCode}
                    onChange={e => setScaleCode(normalizeScaleCodeInput(e.target.value))}
                    maxLength={7} inputMode="numeric"
                    className={inputClass}
                  />
                  <p className="text-2xs text-ink-4 mt-1">
                    The full code programmed into the scale for this product, including its prefix (e.g. 2900633) — type it exactly as shown on the scale, leave blank if it has a normal fixed barcode.
                  </p>
                </div>
                <div className="grid grid-cols-2 gap-3 mt-2">
                  <div>
                    <label htmlFor="product-pricing-mode" className={labelClass}>Pricing</label>
                    <select
                      id="product-pricing-mode" value={pricingMode}
                      onChange={e => setPricingMode(e.target.value as SKUPricingMode)}
                      className={inputClass}
                    >
                      <option value="fixed">Fixed — same price per package (or a normal per-unit product)</option>
                      <option value="per_weight">Per kilo — price = rate × scanned weight</option>
                    </select>
                    {pricingMode === 'per_weight' && !scaleCode && (
                      <p className="text-2xs text-warning mt-1">
                        Set a scale code above too — per-kilo pricing only applies once a package's weight can actually be scanned.
                      </p>
                    )}
                  </div>
                  {pricingMode === 'per_weight' && (
                    <div>
                      <label htmlFor="product-price-per-kilo" className={labelClass}>Price per kilo ({currency})</label>
                      <input
                        id="product-price-per-kilo"
                        type="number" step="0.01" min="0" required value={pricePerKilo}
                        onChange={e => setPricePerKilo(e.target.value)}
                        className={inputClass}
                      />
                    </div>
                  )}
                </div>
                {scaleCode && currentSku && <div className="mt-2"><ScaleCodeAliasesEditor skuId={currentSku.id} /></div>}
              </FormCard>
            )}

            {showFlatSKU && (
              <FormCard
                title="Weight pool"
                description="Book a bulk lot once (e.g. 20kg), then peel off individually-weighed sellable items from it over time as they're packaged."
              >
                <div>
                  <label htmlFor="product-unit-type" className={labelClass}>Item type</label>
                  <select
                    id="product-unit-type" value={unitType}
                    onChange={e => {
                      const next = e.target.value as SKUUnitType
                      setUnitType(next)
                      if (next === 'kilo') { setPortionedFromSKUID(null); setPortionedFromLabel('') }
                    }}
                    className={inputClass}
                  >
                    <option value="unit">Unit</option>
                    <option value="box">Box</option>
                    <option value="kilo">Kilo (weight pool)</option>
                  </select>
                </div>
                {unitType !== 'kilo' && (
                  portionedFromSKUID ? (
                    <div className="flex items-center gap-2 mt-2">
                      <span className="flex-1 text-xs text-ink">
                        Portioned from <span className="font-medium">{portionedFromLabel || portionedFromSKUID}</span>
                      </span>
                      <button
                        type="button"
                        onClick={() => { setPortionedFromSKUID(null); setPortionedFromLabel('') }}
                        className="text-2xs text-ink-4 hover:underline"
                      >
                        Remove
                      </button>
                    </div>
                  ) : portionPickerOpen ? (
                    <div className="mt-2">
                      <CaseOfSKUPicker
                        excludeSKUID={currentSku?.id ?? ''}
                        filterSku={s => s.unit_type === 'kilo'}
                        onSelect={(sku, label) => { setPortionedFromSKUID(sku.id); setPortionedFromLabel(label); setPortionPickerOpen(false) }}
                        onClose={() => setPortionPickerOpen(false)}
                      />
                    </div>
                  ) : (
                    <button
                      type="button"
                      onClick={() => setPortionPickerOpen(true)}
                      className="flex items-center gap-1 text-xs font-medium text-accent hover:text-accent/80 transition-colors mt-2"
                    >
                      <Search size={12} /> This is portioned from a weight pool
                    </button>
                  )
                )}
              </FormCard>
            )}

            {showFlatSKU && (
              <FormCard title="Shipping" description="Leave off for digital/virtual products.">
                <ToggleRow label="Virtual product (no shipping)" checked={isVirtual} onChange={() => setIsVirtual(v => !v)} />
                {!isVirtual && (
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label htmlFor="product-weight" className={labelClass}>Weight (g)</label>
                      <input
                        id="product-weight"
                        type="number" step="1" min="0" placeholder="Optional" value={weightGrams} onChange={e => setWeightGrams(e.target.value)}
                        className={inputClass}
                      />
                    </div>
                    <div>
                      <label htmlFor="product-length" className={labelClass}>Length (mm)</label>
                      <input
                        id="product-length"
                        type="number" step="1" min="0" placeholder="Optional" value={lengthMM} onChange={e => setLengthMM(e.target.value)}
                        className={inputClass}
                      />
                    </div>
                    <div>
                      <label htmlFor="product-width" className={labelClass}>Width (mm)</label>
                      <input
                        id="product-width"
                        type="number" step="1" min="0" placeholder="Optional" value={widthMM} onChange={e => setWidthMM(e.target.value)}
                        className={inputClass}
                      />
                    </div>
                    <div>
                      <label htmlFor="product-height" className={labelClass}>Height (mm)</label>
                      <input
                        id="product-height"
                        type="number" step="1" min="0" placeholder="Optional" value={heightMM} onChange={e => setHeightMM(e.target.value)}
                        className={inputClass}
                      />
                    </div>
                  </div>
                )}
              </FormCard>
            )}

          </div>

          {/* Sidebar column */}
          <div className="space-y-5">
            <div className="rounded-xl border border-line p-4 divide-y divide-line/60">
              <div className="pb-4">
                <label htmlFor="product-status-sidebar" className={labelClass}>Status</label>
                <select
                  id="product-status-sidebar"
                  value={status} onChange={e => setStatus(e.target.value as ProductStatus)}
                  className={inputClass}
                >
                  <option value="draft">Draft</option>
                  <option value="active">Active</option>
                  <option value="archived">Archived</option>
                </select>
              </div>

              <div className="pt-4 pb-4">
                <label htmlFor="product-category" className={labelClass}>Category</label>
                <select
                  id="product-category"
                  value={categoryId} onChange={e => setCategoryId(e.target.value)}
                  className={inputClass}
                >
                  <option value="">No category</option>
                  {sortCategoriesHierarchically(categories).map(c => (
                    <option key={c.id} value={c.id}>{'    '.repeat(c.depth)}{c.depth > 0 ? '↳ ' : ''}{c.name}</option>
                  ))}
                </select>
                <div className="flex gap-1.5 mt-1.5">
                  <input
                    type="text" value={newCategoryName} onChange={e => setNewCategoryName(e.target.value)}
                    placeholder="+ new category name"
                    className="flex-1 px-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
                  />
                  <button type="button" onClick={handleAddCategory} className="px-3 py-1.5 rounded-lg dark:bg-night-800 bg-gray-100 text-2xs font-medium text-ink-2 hover:bg-hover transition-colors">
                    Add
                  </button>
                </div>
                {categoryId && (
                  <CategoryMarginEditor
                    category={categories.find(c => c.id === categoryId) ?? null}
                    onSave={margin => onUpdateCategory(categoryId, { name: categories.find(c => c.id === categoryId)!.name, target_markup_basis_points: margin })}
                  />
                )}
              </div>

              <div className="pt-4">
                <label htmlFor="product-vendor" className={labelClass}>Vendor</label>
                <input id="product-vendor" disabled placeholder="Coming soon" className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
              </div>

              <div className="pt-4">
                <label htmlFor="product-collections" className={labelClass}>Collections</label>
                <input id="product-collections" disabled placeholder="Coming soon" className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
              </div>
            </div>

            <FormCard title="SEO" description="Coming soon.">
              <div>
                <label htmlFor="product-seo-title" className={labelClass}>Page title</label>
                <input id="product-seo-title" disabled placeholder="Coming soon" className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
              </div>
              <div>
                <label htmlFor="product-seo-description" className={labelClass}>Meta description</label>
                <textarea id="product-seo-description" disabled placeholder="Coming soon" rows={2} className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
              </div>
            </FormCard>
          </div>
        </div>
      </form>
    )
  }
}
