import { useState, useEffect, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  Plus, Search, Package, Trash2, Edit3, Copy, ArrowLeft, Plug, X, ImagePlus, Boxes, RefreshCw,
} from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useApp } from '../context/AppContext'
import Pagination from '../components/common/Pagination'
import { useProducts } from '../pms/hooks/useProducts'
import { useCategories } from '../pms/hooks/useCategories'
import { sortCategoriesHierarchically } from '../pms/categoryTree'
import { useProductDetail } from '../pms/hooks/useProductDetail'
import { useProductMedia } from '../pms/hooks/useProductMedia'
import { ProductBolTab } from '../pms/ProductBolTab'
import * as pmsApi from '../pms/api'
import * as integrationsApi from '../integrations/api'
import * as inventoryApi from '../inventory/api'
import type { Connection, ProductChannelListing, ProductChannelListingSyncStatus } from '../integrations/types'
import type { Category, Product, ProductStatus, SKUStatus, SKUUnitType, VariantWithSKUs, Media, SKU, SKUMarginInsight, SKUPriceHistoryEntry, SKUPriceHistoryReason } from '../pms/types'
import type { StockLevel } from '../inventory/types'
import { useTaxClasses } from '../tax/hooks/useTaxClasses'
import { useTaxRates } from '../tax/hooks/useTaxRates'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'

const STATUS_STYLES: Record<ProductStatus, string> = {
  active: 'bg-green-500/15 text-green-400',
  archived: 'bg-gray-500/15 text-gray-400',
  draft: 'bg-amber-500/15 text-amber-400',
}

const SKU_STATUS_STYLES: Record<SKUStatus, string> = {
  active: 'bg-green-500/15 text-green-400',
  inactive: 'bg-gray-500/15 text-gray-400',
}

const CURRENCIES = ['USD', 'EUR', 'GBP', 'CAD', 'AUD']
const PAGE_SIZE = 20

function formatPrice(minorUnits: number, currency: string): string {
  return `${minorUnits < 0 ? '-' : ''}${currency} ${(Math.abs(minorUnits) / 100).toFixed(2)}`
}

function parsePriceToMinorUnits(price: string): number {
  return Math.round(parseFloat(price || '0') * 100)
}

function parseOptionalPriceToMinorUnits(price: string): number | null {
  if (price.trim() === '') return null
  return Math.round(parseFloat(price) * 100)
}

function formatMinorUnitsAsInput(minorUnits: number): string {
  return (minorUnits / 100).toFixed(2)
}

function parseIntOrUndefined(value: string): number | undefined {
  if (value === '') return undefined
  const n = parseInt(value, 10)
  return Number.isNaN(n) ? undefined : n
}

export default function Products() {
  const { token } = useAuth()

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)

  // The URL is the source of truth for "which product is open" (?id=<uuid>,
  // or ?id=new for the create flow) -- survives a refresh, and doubles as
  // the "re-clicking Products in the sidebar returns to the list" fix:
  // Sidebar's setActiveNav navigates to the plain "/products" path (see
  // AppContext), which has no id param, so editingId below goes back to
  // null on its own -- no separate reset-on-reclick mechanism needed.
  const [searchParams, setSearchParams] = useSearchParams()
  const editingId = searchParams.get('id')
  const openEditor = (id: string) => setSearchParams({ id })
  const closeEditor = () => setSearchParams({})

  // Debounce the search box so it doesn't fire a server request on every
  // keystroke — the list is now fetched server-side, unlike the old
  // client-side .filter() over a fully-loaded array. Only commits
  // setSearch, never setPage directly -- the separate effect below resets
  // the page, keyed off `search` actually changing. Doing it in one step
  // here meant this fired ~300ms after every mount regardless of whether
  // the search box was touched (searchInput debounces to itself), so a
  // page the user had already navigated to within that window got
  // silently reset back to 1.
  useEffect(() => {
    const timeout = setTimeout(() => setSearch(searchInput), 300)
    return () => clearTimeout(timeout)
  }, [searchInput])

  useEffect(() => {
    setPage(1)
  }, [search])

  const { products, total, loading, error, create: createProduct, update: updateProduct, remove: removeProduct } =
    useProducts(token ?? '', { page, perPage: PAGE_SIZE, search })
  const { categories, create: createCategory, update: updateCategory } = useCategories(token ?? '')

  const categoryName = (id?: string | null) => categories.find(c => c.id === id)?.name ?? '—'

  const [duplicatingId, setDuplicatingId] = useState<string | null>(null)
  const [duplicateError, setDuplicateError] = useState('')
  const handleDuplicate = async (id: string) => {
    if (!token) return
    setDuplicatingId(id)
    setDuplicateError('')
    try {
      const detail = await pmsApi.duplicateProduct(token, id)
      openEditor(detail.product.id)
    } catch (err) {
      setDuplicateError(err instanceof Error ? err.message : 'Failed to duplicate product')
    } finally {
      setDuplicatingId(null)
    }
  }

  // Swaps ?id=new for the real id the instant creation succeeds -- without
  // this, a refresh in the gap between "create" and the next explicit
  // navigation would reopen a blank "New Product" form instead of the
  // product that was just created (it's safely persisted either way; only
  // the "which product is open" URL state would be stale).
  const handleCreateProduct = async (payload: Parameters<typeof createProduct>[0]) => {
    const created = await createProduct(payload)
    openEditor(created.id)
    return created
  }

  if (editingId !== null) {
    return (
      <ProductEditPage
        productId={editingId === 'new' ? null : editingId}
        categories={categories}
        onCreateCategory={createCategory}
        onUpdateCategory={updateCategory}
        onCreate={handleCreateProduct}
        onUpdate={updateProduct}
        onClose={closeEditor}
      />
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
          <input
            value={searchInput}
            onChange={e => setSearchInput(e.target.value)}
            placeholder="Search by name or slug..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white dark:border-[#2D3748] border-gray-200 border dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <button
            onClick={() => openEditor('new')}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 transition-colors font-medium"
          >
            <Plus size={13} /> New Product
          </button>
        </div>
      </div>

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">{total} product{total === 1 ? '' : 's'}</span>
        {(error || duplicateError) && <span className="text-xs text-red-400 ml-auto">{error || duplicateError}</span>}
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Product</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Category</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Price</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Sale Price</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 w-8" />
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={7} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && products.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No products yet</td></tr>
            )}
            {products.map(p => (
              <tr
                key={p.id}
                onClick={() => openEditor(p.id)}
                className="group hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  {p.image_url ? (
                    <img src={p.image_url} alt="" className="w-6 h-6 rounded object-cover dark:bg-[#2D3748] bg-gray-200" />
                  ) : (
                    <div className="w-6 h-6 rounded dark:bg-[#2D3748] bg-gray-200 flex items-center justify-center">
                      <Package size={12} className="dark:text-gray-500 text-gray-400" />
                    </div>
                  )}
                </td>
                <td className="px-3 py-2.5 max-w-[220px]">
                  <p className="font-medium dark:text-gray-100 text-gray-900 truncate">{p.name}</p>
                  <p className="text-[10px] font-mono dark:text-gray-500 text-gray-400 truncate">{p.slug}</p>
                </td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{categoryName(p.category_id)}</td>
                <td className="px-3 py-2.5 dark:text-gray-300 text-gray-700">
                  {p.price_minor_units != null ? formatPrice(p.price_minor_units, p.currency_code ?? '') : '—'}
                </td>
                <td className="px-3 py-2.5">
                  {p.sale_price_minor_units != null ? (
                    <span className="text-red-400 font-medium">{formatPrice(p.sale_price_minor_units, p.currency_code ?? '')}</span>
                  ) : (
                    <span className="dark:text-gray-600 text-gray-300">—</span>
                  )}
                </td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium capitalize', STATUS_STYLES[p.status])}>
                    {p.status}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={e => { e.stopPropagation(); openEditor(p.id) }}
                      className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors"
                      title="Edit"
                    >
                      <Edit3 size={12} className="dark:text-gray-400 text-gray-500" />
                    </button>
                    <button
                      onClick={e => { e.stopPropagation(); handleDuplicate(p.id) }}
                      disabled={duplicatingId === p.id}
                      className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors disabled:opacity-50"
                      title="Duplicate"
                    >
                      <Copy size={12} className="dark:text-gray-400 text-gray-500" />
                    </button>
                    <button
                      onClick={e => {
                        e.stopPropagation()
                        if (window.confirm(`Delete "${p.name}"? This also deletes its variants and SKUs.`)) {
                          removeProduct(p.id)
                        }
                      }}
                      className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors"
                      title="Delete"
                    >
                      <Trash2 size={12} className="dark:text-gray-400 text-gray-500 hover:text-red-400" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
    </div>
  )
}

// ── Product Edit Page ────────────────────────────────────────────────────────
//
// Dedicated full page (not a slide-over) for creating/editing a product.
// "General" tab holds the shared data every provider needs (name, price,
// variants...). Additional tabs hold provider-specific values — only
// WooCommerce for now, one tab per connected integration eventually
// (Shopify, Lightspeed, ...), each mapping the same product to that
// channel's own fields without polluting the generic model.

type EditorTab = 'general' | 'woocommerce' | 'bol'

interface ProductEditorProps {
  productId: string | null
  categories: Category[]
  onCreateCategory: (payload: { name: string }) => Promise<{ id: string; name: string }>
  onUpdateCategory: (id: string, payload: { name: string; parent_id?: string | null; target_markup_basis_points?: number | null }) => Promise<Category>
  onCreate: (payload: { name: string; description?: string; category_id?: string | null; status: ProductStatus; has_variants?: boolean; sync_variants_as_woo_variations?: boolean; tags?: string[] }) => Promise<Product>
  onUpdate: (id: string, payload: { name: string; slug?: string; description?: string; category_id?: string | null; status: ProductStatus; has_variants?: boolean; sync_variants_as_woo_variations?: boolean; tags?: string[] }) => Promise<Product>
  onClose: () => void
}

function ProductEditPage({ productId, categories, onCreateCategory, onUpdateCategory, onCreate, onUpdate, onClose }: ProductEditorProps) {
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
  const [activeTab, setActiveTab] = useState<EditorTab>('general')
  const [resyncing, setResyncing] = useState(false)
  const [resyncMessage, setResyncMessage] = useState('')

  // Integration tabs only ever appear for a provider that's actually
  // connected — an unconnected/disabled integration gets no tab at all,
  // rather than a tab that just tells you to go connect it.
  const [wooConnection, setWooConnection] = useState<Connection | null>(null)
  const [bolConnection, setBolConnection] = useState<Connection | null>(null)
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
        setSavedProductId(product.id)
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
              tax_class_id: taxClassId,
              status: skuStatus,
            })
          }
        }
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

  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500 transition-shadow'
  const labelClass = 'block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Products
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">
            {savedProductId ? name || 'Edit Product' : 'New Product'}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          {resyncMessage && <span className="text-[11px] dark:text-gray-500 text-gray-400 mr-1">{resyncMessage}</span>}
          {savedProductId && wooConnection && (
            <button
              type="button" onClick={handleResync} disabled={resyncing}
              title="Force-push this product's current fields and stock to every linked channel, even if nothing changed"
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs font-semibold dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50 disabled:opacity-50 transition-colors"
            >
              <RefreshCw size={12} className={resyncing ? 'animate-spin' : undefined} />
              {resyncing ? 'Resyncing…' : 'Resync to channels'}
            </button>
          )}
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs font-semibold dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50 transition-colors"
          >
            Discard
          </button>
          <button
            type="submit" form="product-general-form" disabled={saving || activeTab !== 'general'}
            className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : savedProductId ? 'Save' : 'Create Product'}
          </button>
        </div>
      </div>

      <div className="flex items-center gap-1 px-5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <button
          onClick={() => setActiveTab('general')}
          className={clsx(
            'px-3 py-2.5 text-xs font-medium border-b-2 -mb-px transition-colors',
            activeTab === 'general'
              ? 'border-blue-500 text-blue-400'
              : 'border-transparent dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600',
          )}
        >
          General
        </button>
        {savedProductId && wooConnection && (
          <button
            onClick={() => setActiveTab('woocommerce')}
            className={clsx(
              'flex items-center gap-1.5 px-3 py-2.5 text-xs font-medium border-b-2 -mb-px transition-colors',
              activeTab === 'woocommerce'
                ? 'border-blue-500 text-blue-400'
                : 'border-transparent dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600',
            )}
          >
            <Plug size={12} /> WooCommerce
          </button>
        )}
        {savedProductId && bolConnection && (
          <button
            onClick={() => setActiveTab('bol')}
            className={clsx(
              'flex items-center gap-1.5 px-3 py-2.5 text-xs font-medium border-b-2 -mb-px transition-colors',
              activeTab === 'bol'
                ? 'border-blue-500 text-blue-400'
                : 'border-transparent dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600',
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
          <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-3 py-2">{formError}</p>
        </div>
      )}

      <div className="flex-1 overflow-y-auto">
        <div className={clsx('max-w-5xl mx-auto p-6', activeTab !== 'general' && 'hidden')}>
          {renderGeneralTab()}
        </div>
        {activeTab === 'woocommerce' && savedProductId && wooConnection && (
          <div className="max-w-5xl mx-auto p-6">
            <ProductWooCommerceTab productId={savedProductId} connection={wooConnection} sku={currentSku} />
          </div>
        )}
        {activeTab === 'bol' && savedProductId && bolConnection && (
          <div className="max-w-5xl mx-auto p-6">
            <ProductBolTab productId={savedProductId} connection={bolConnection} sku={currentSku} />
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
                      className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] dark:bg-[#2D3748] bg-gray-100 dark:text-gray-300 text-gray-700"
                    >
                      {tag}
                      <button type="button" onClick={() => removeTag(tag)} aria-label={`Remove tag ${tag}`}>
                        <X size={10} className="dark:text-gray-500 text-gray-400" />
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
                    className="flex-1 min-w-[6rem] bg-transparent outline-none text-xs dark:text-gray-300 text-gray-700"
                  />
                </div>
                <p className="mt-1.5 text-[10px] dark:text-gray-500 text-gray-400">
                  Press Enter or comma to add. Used by Sales Channels' "by tag" enrollment rule.
                </p>
              </div>
            </FormCard>

            <FormCard title="Media">
              {savedProductId ? (
                <MediaSection productId={savedProductId} />
              ) : (
                <p className="text-[11px] dark:text-gray-500 text-gray-400">Save the product first, then add images.</p>
              )}
            </FormCard>

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
                      <span className="text-[11px] dark:text-gray-400 text-gray-600">
                        <span className="font-medium dark:text-gray-300 text-gray-800">Sync variants as WooCommerce variations</span>
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
                    <p className="text-[11px] dark:text-gray-500 text-gray-400">Save the product first, then add variants below.</p>
                  )}
                  {!savedProductId && (
                    <button
                      type="button"
                      onClick={() => setVariantsMode(false)}
                      className="text-[11px] dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors"
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
                    className="text-xs font-medium text-blue-400 hover:text-blue-300 transition-colors disabled:opacity-50"
                  >
                    {enablingVariants ? 'Enabling…' : '+ Add variants like size or color'}
                  </button>
                  {enableVariantsError && <p className="text-[11px] text-red-500 mt-1.5">{enableVariantsError}</p>}
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
                      type="text" placeholder="Optional" value={barcode} onChange={e => setBarcode(e.target.value)}
                      className={inputClass}
                    />
                  </div>
                  <div>
                    <label htmlFor="product-price" className={labelClass}>Price</label>
                    <input
                      id="product-price"
                      type="number" step="0.01" min="0" required={!variantsMode} placeholder="0.00" value={price} onChange={e => setPrice(e.target.value)}
                      className={inputClass}
                    />
                    {taxPreview && (
                      <p className="mt-1 text-[10px] dark:text-gray-500 text-gray-400">
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
                    <span className="flex-1 text-xs dark:text-gray-200 text-gray-800">
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
                      className="text-[10px] dark:text-gray-500 text-gray-400 hover:underline"
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
                    className="flex items-center gap-1 text-xs font-medium text-blue-400 hover:text-blue-300 transition-colors"
                  >
                    <Search size={12} /> This is a case of another product
                  </button>
                )}
              </FormCard>
            )}

            {showFlatSKU && currentSku && (
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
                      <span className="flex-1 text-xs dark:text-gray-200 text-gray-800">
                        Portioned from <span className="font-medium">{portionedFromLabel || portionedFromSKUID}</span>
                      </span>
                      <button
                        type="button"
                        onClick={() => { setPortionedFromSKUID(null); setPortionedFromLabel('') }}
                        className="text-[10px] dark:text-gray-500 text-gray-400 hover:underline"
                      >
                        Remove
                      </button>
                    </div>
                  ) : portionPickerOpen ? (
                    <div className="mt-2">
                      <CaseOfSKUPicker
                        excludeSKUID={currentSku.id}
                        filterSku={s => s.unit_type === 'kilo'}
                        onSelect={(sku, label) => { setPortionedFromSKUID(sku.id); setPortionedFromLabel(label); setPortionPickerOpen(false) }}
                        onClose={() => setPortionPickerOpen(false)}
                      />
                    </div>
                  ) : (
                    <button
                      type="button"
                      onClick={() => setPortionPickerOpen(true)}
                      className="flex items-center gap-1 text-xs font-medium text-blue-400 hover:text-blue-300 transition-colors mt-2"
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
            <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 divide-y dark:divide-[#2D3748]/60 divide-gray-100">
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
                    className="flex-1 px-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none"
                  />
                  <button type="button" onClick={handleAddCategory} className="px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-gray-100 text-[10px] font-medium dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-200 transition-colors">
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

// ── Shared form layout primitives ────────────────────────────────────────────

// MarginInsightCard shows the cost price the daily cron computed, the
// recommended price from the effective target margin, and profit/margin at
// whatever price is currently set -- see application/pms's
// GetSKUMarginInsightUseCase. Fetched fresh whenever skuId changes (e.g.
// switching between products), not tied to the price/salePrice form state --
// this reflects the *saved* SKU, not an in-progress unsaved edit.
// btwVariant computes profit/margin/markup at a given selling price against
// a fixed cost -- used to derive the "with BTW" / "without BTW" pair from
// whichever of the two the stored price already represents (per
// tenant.prices_include_tax), by converting the price with taxRateBasisPoints
// and re-deriving profit from there. Cost never carries tax (what you paid
// a supplier), so only price/profit shift between the two variants.
function btwVariant(priceMinorUnits: number, costMinorUnits: number) {
  const profit = priceMinorUnits - costMinorUnits
  return {
    price: priceMinorUnits,
    profit,
    marginBasisPoints: priceMinorUnits > 0 ? Math.round((profit * 10000) / priceMinorUnits) : 0,
    markupBasisPoints: costMinorUnits > 0 ? Math.round((profit * 10000) / costMinorUnits) : 0,
  }
}

function MarginInsightCard({ skuId, currency, taxRateBasisPoints, pricesIncludeTax }: {
  skuId: string; currency: string; taxRateBasisPoints: number | null; pricesIncludeTax: boolean
}) {
  const { token } = useAuth()
  const [insight, setInsight] = useState<SKUMarginInsight | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const reload = () => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError('')
    pmsApi.getSKUMarginInsight(token, skuId)
      .then(result => { if (!cancelled) setInsight(result) })
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load margin insight') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }

  useEffect(reload, [token, skuId])

  const money = (minorUnits: number) => formatPrice(minorUnits, currency)
  const pct = (basisPoints: number) => `${(basisPoints / 100).toFixed(1)}%`

  // withBoth derives the excl./incl.-BTW pair for one selling price, given
  // the rate and which side the stored price already represents.
  const withBoth = (sellingPriceMinorUnits: number, costMinorUnits: number) => {
    if (taxRateBasisPoints == null) return null
    const rate = taxRateBasisPoints / 10000
    const exclusivePrice = pricesIncludeTax ? Math.round(sellingPriceMinorUnits / (1 + rate)) : sellingPriceMinorUnits
    const inclusivePrice = pricesIncludeTax ? sellingPriceMinorUnits : Math.round(sellingPriceMinorUnits * (1 + rate))
    return { exclusive: btwVariant(exclusivePrice, costMinorUnits), inclusive: btwVariant(inclusivePrice, costMinorUnits) }
  }

  const profitRow = (label: string, profitMinorUnits: number, marginBps: number, markupBps: number, sellingPriceMinorUnits: number, costMinorUnits: number) => {
    const both = withBoth(sellingPriceMinorUnits, costMinorUnits)
    return (
      <div>
        <div className="dark:text-gray-500 text-gray-400">{label}</div>
        <div className={clsx('font-medium', profitMinorUnits < 0 ? 'text-red-400' : 'dark:text-gray-100 text-gray-900')}>
          {money(profitMinorUnits)} <span className="font-normal">({pct(marginBps)} margin / {pct(markupBps)} markup)</span>
        </div>
        {both && (
          <div className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">
            Excl. BTW: {money(both.exclusive.profit)} ({pct(both.exclusive.marginBasisPoints)} margin / {pct(both.exclusive.markupBasisPoints)} markup)
            {' · '}
            Incl. BTW: {money(both.inclusive.profit)} ({pct(both.inclusive.marginBasisPoints)} margin / {pct(both.inclusive.markupBasisPoints)} markup)
          </div>
        )}
      </div>
    )
  }

  return (
    <FormCard title="Cost & margin insight" description="Cost price is recomputed daily from purchase order receiving history; not a live figure.">
      {loading && <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
      {!loading && error && <p className="text-xs text-red-400">{error}</p>}
      {!loading && !error && insight && !insight.cost_price_available && (
        <p className="text-xs dark:text-gray-500 text-gray-400">
          No cost price yet — this SKU hasn't been received against a purchase order, or the daily cost-price job hasn't run since.
        </p>
      )}
      {!loading && !error && insight && insight.cost_price_available && (
        <div className="grid grid-cols-2 gap-x-4 gap-y-3 text-xs">
          <div>
            <div className="dark:text-gray-500 text-gray-400">Cost price</div>
            <div className="font-medium dark:text-gray-100 text-gray-900">{money(insight.cost_price_minor_units)}</div>
          </div>
          <div>
            <div className="dark:text-gray-500 text-gray-400">Recommended price</div>
            <div className="font-medium dark:text-gray-100 text-gray-900">
              {money(insight.recommended_price_minor_units)}
              <span className="dark:text-gray-500 text-gray-400 font-normal"> ({pct(insight.effective_target_markup_basis_points)} target markup)</span>
            </div>
          </div>
          {profitRow(
            'Profit at current price', insight.profit_at_price_minor_units, insight.margin_at_price_basis_points, insight.markup_at_price_basis_points,
            insight.cost_price_minor_units + insight.profit_at_price_minor_units, insight.cost_price_minor_units,
          )}
          {insight.profit_at_sale_price_minor_units != null && insight.margin_at_sale_price_basis_points != null && insight.markup_at_sale_price_basis_points != null && (
            profitRow(
              'Profit at sale price', insight.profit_at_sale_price_minor_units, insight.margin_at_sale_price_basis_points, insight.markup_at_sale_price_basis_points,
              insight.cost_price_minor_units + insight.profit_at_sale_price_minor_units, insight.cost_price_minor_units,
            )
          )}
          <div>
            <div className="dark:text-gray-500 text-gray-400">On hand</div>
            <div className="font-medium dark:text-gray-100 text-gray-900">{insight.total_on_hand_quantity} units</div>
          </div>
          <div>
            <div className="dark:text-gray-500 text-gray-400">Total profit potential</div>
            <div className="font-medium dark:text-gray-100 text-gray-900">{money(insight.total_profit_potential_minor_units)}</div>
          </div>
          {insight.shrinkage_minor_units > 0 && (
            <div className="col-span-2">
              <div className="dark:text-gray-500 text-gray-400">Shrinkage (damaged/lost, last 30 days)</div>
              <div className="font-medium text-amber-400">{money(insight.shrinkage_minor_units)}</div>
            </div>
          )}
        </div>
      )}
      {!loading && !error && insight && (
        <ManualCostPriceEditor skuId={skuId} insight={insight} currency={currency} onChanged={reload} />
      )}
    </FormCard>
  )
}

const PRICE_HISTORY_REASON_LABEL: Record<SKUPriceHistoryReason, string> = {
  created: 'Created',
  manual_update: 'Manual edit',
  sale_expired: 'Sale expired',
}

// PriceHistoryCard shows the SKU's immutable effective-dated price/sale
// price intervals -- see domain/pms.SKUPriceHistory. A row is only written
// when price/sale price actually changes, so this reconstructs what a SKU
// actually cost at any past point in time, independent of its current
// PriceMinorUnits/SalePriceMinorUnits. Fetched fresh whenever skuId
// changes, same reasoning as MarginInsightCard.
function PriceHistoryCard({ skuId, currency }: { skuId: string; currency: string }) {
  const { token } = useAuth()
  const [history, setHistory] = useState<SKUPriceHistoryEntry[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError('')
    pmsApi.listSKUPriceHistory(token, skuId)
      .then(result => { if (!cancelled) setHistory(result) })
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load price history') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, skuId])

  const money = (minorUnits: number) => formatPrice(minorUnits, currency)
  const formatDate = (value: string) => new Date(value).toLocaleString()

  return (
    <FormCard title="Price history" description="An immutable record of every effective price change -- not the same as the daily-computed cost price above.">
      {loading && <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
      {!loading && error && <p className="text-xs text-red-400">{error}</p>}
      {!loading && !error && history && history.length === 0 && (
        <p className="text-xs dark:text-gray-500 text-gray-400">No price changes recorded yet.</p>
      )}
      {!loading && !error && history && history.length > 0 && (
        <table className="w-full text-xs">
          <thead>
            <tr className="dark:text-gray-500 text-gray-400">
              <th className="text-left font-medium pb-1.5 pr-3">Valid from</th>
              <th className="text-left font-medium pb-1.5 pr-3">Valid until</th>
              <th className="text-left font-medium pb-1.5 pr-3">Price</th>
              <th className="text-left font-medium pb-1.5 pr-3">Sale price</th>
              <th className="text-left font-medium pb-1.5">Reason</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {history.map(entry => (
              <tr key={entry.id}>
                <td className="py-1.5 pr-3 dark:text-gray-300 text-gray-700">{formatDate(entry.valid_from)}</td>
                <td className="py-1.5 pr-3 dark:text-gray-300 text-gray-700">
                  {entry.valid_until ? formatDate(entry.valid_until) : <span className="text-green-400 font-medium">Current</span>}
                </td>
                <td className="py-1.5 pr-3 dark:text-gray-100 text-gray-900 font-medium">{money(entry.price_minor_units)}</td>
                <td className="py-1.5 pr-3 dark:text-gray-100 text-gray-900">
                  {entry.sale_price_minor_units != null ? money(entry.sale_price_minor_units) : '—'}
                </td>
                <td className="py-1.5 dark:text-gray-400 text-gray-500">{PRICE_HISTORY_REASON_LABEL[entry.reason]}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </FormCard>
  )
}

// ManualCostPriceEditor lets staff set a SKU's cost price by hand -- for a
// SKU the daily cron has no basis to compute at all (e.g. a purchase order
// marked received without ever creating a batch), or to correct a value
// staff know is wrong. Setting one sticks until "Revert to automatic" is
// pressed -- the cron skips any SKU with a manual override entirely (see
// application/pms's RecalculateCostPricesUseCase).
function ManualCostPriceEditor({ skuId, insight, currency, onChanged }: { skuId: string; insight: SKUMarginInsight; currency: string; onChanged: () => void }) {
  const { token } = useAuth()
  const [value, setValue] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const isManual = insight.cost_price_source === 'manual'

  // Prefill from the computed cost price when one exists, so overriding it
  // is a tweak starting from the real value rather than blank re-entry --
  // only a SKU with no cost basis at all starts empty.
  useEffect(() => {
    if (isManual) return
    setValue(insight.cost_price_available ? (insight.cost_price_minor_units / 100).toFixed(2) : '')
  }, [skuId, isManual, insight.cost_price_available, insight.cost_price_minor_units])

  const handleSet = async () => {
    if (!token || value.trim() === '') return
    setSaving(true)
    setError('')
    try {
      await pmsApi.setManualCostPrice(token, skuId, parsePriceToMinorUnits(value))
      setValue('')
      onChanged()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to set cost price')
    } finally {
      setSaving(false)
    }
  }

  const handleClear = async () => {
    if (!token) return
    setSaving(true)
    setError('')
    try {
      await pmsApi.clearCostPriceOverride(token, skuId)
      onChanged()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to revert cost price')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="pt-3 border-t dark:border-[#2D3748] border-gray-200">
      {isManual ? (
        <div className="flex items-center gap-2 text-xs">
          <span className="dark:text-gray-500 text-gray-400">
            Cost price set manually ({currency} {(insight.cost_price_minor_units / 100).toFixed(2)}) -- the daily job won't overwrite it.
          </span>
          <button
            type="button" onClick={handleClear} disabled={saving}
            className="px-2 py-1 rounded-lg dark:bg-[#1F2937] bg-gray-100 text-[10px] font-medium dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-200 transition-colors disabled:opacity-50 shrink-0"
          >
            {saving ? 'Reverting…' : 'Revert to automatic'}
          </button>
        </div>
      ) : (
        <div className="flex items-center gap-1.5">
          <span className="text-[10px] dark:text-gray-500 text-gray-400 shrink-0">
            {insight.cost_price_available ? 'Override cost price' : 'Set cost price manually'}
          </span>
          <input
            type="number" step="0.01" min="0" placeholder={currency}
            value={value} onChange={e => setValue(e.target.value)}
            className="w-24 px-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none"
          />
          <button
            type="button" onClick={handleSet} disabled={saving || value.trim() === ''}
            className="px-2 py-1 rounded-lg dark:bg-[#1F2937] bg-gray-100 text-[10px] font-medium dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-200 transition-colors disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Set'}
          </button>
        </div>
      )}
      {error && <p className="text-[10px] text-red-400 mt-1">{error}</p>}
    </div>
  )
}

// CategoryMarginEditor lets staff override the tenant-wide default target
// markup for every product in the selected category (see
// pms.Category.TargetMarkupBasisPoints) -- an empty input means "use the
// tenant default", matching UpdateCategoryInput's full-replace semantics.
function CategoryMarginEditor({ category, onSave }: { category: Category | null; onSave: (basisPoints: number | null) => Promise<Category> }) {
  const [value, setValue] = useState('')
  const [saving, setSaving] = useState(false)
  const [savedMsg, setSavedMsg] = useState(false)

  useEffect(() => {
    setValue(category?.target_markup_basis_points != null ? (category.target_markup_basis_points / 100).toString() : '')
    setSavedMsg(false)
  }, [category?.id, category?.target_markup_basis_points])

  if (!category) return null

  const handleSave = async () => {
    setSaving(true)
    try {
      const basisPoints = value.trim() === '' ? null : Math.round(parseFloat(value) * 100)
      await onSave(basisPoints)
      setSavedMsg(true)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex items-center gap-1.5 mt-1.5">
      <input
        type="number" step="0.1" min="0" placeholder="Tenant default"
        value={value} onChange={e => { setValue(e.target.value); setSavedMsg(false) }}
        className="w-28 px-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none"
      />
      <span className="text-[10px] dark:text-gray-500 text-gray-400">% target markup for this category</span>
      <button type="button" onClick={handleSave} disabled={saving} className="px-2 py-1 rounded-lg dark:bg-[#1F2937] bg-gray-100 text-[10px] font-medium dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-200 transition-colors disabled:opacity-50">
        {saving ? 'Saving…' : 'Save'}
      </button>
      {savedMsg && <span className="text-[10px] text-green-400">Saved</span>}
    </div>
  )
}

function FormCard({ title, description, children }: { title: string; description?: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
      <div>
        <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">{title}</h3>
        {description && <p className="text-[11px] dark:text-gray-500 text-gray-400 mt-0.5">{description}</p>}
      </div>
      {children}
    </div>
  )
}

function ToggleRow({ label, checked, onChange }: { label: string; checked: boolean; onChange: () => void }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-xs font-medium dark:text-gray-200 text-gray-800">{label}</span>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        onClick={onChange}
        className={clsx(
          'relative w-9 h-5 rounded-full shrink-0 transition-colors',
          checked ? 'bg-blue-600' : 'dark:bg-[#2D3748] bg-gray-300',
        )}
      >
        <span className={clsx(
          'absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform',
          checked ? 'translate-x-4' : 'translate-x-0',
        )} />
      </button>
    </div>
  )
}

// ── WooCommerce tab ───────────────────────────────────────────────────────────
//
// Per-channel sync state for this product on the connected WooCommerce
// store, fetched from GetProductChannelListingUseCase. A 404 (no listing
// row) is an expected, common state — most products aren't linked to every
// connected channel — rendered as "not linked yet", not an error.

const WOO_SYNC_STATUS_LABEL: Record<ProductChannelListingSyncStatus, string> = {
  pending: 'Pending', synced: 'Synced', error: 'Error',
}
const WOO_SYNC_STATUS_STYLE: Record<ProductChannelListingSyncStatus, string> = {
  pending: 'dark:text-amber-400 text-amber-600',
  synced: 'dark:text-green-400 text-green-600',
  error: 'dark:text-red-400 text-red-600',
}

function ProductWooCommerceTab({ productId, connection, sku }: { productId: string; connection: Connection; sku: SKU | null }) {
  const { token } = useAuth()
  const [listing, setListing] = useState<ProductChannelListing | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError(null)
    integrationsApi.getProductChannelListing(token, connection.id, productId)
      .then(result => { if (!cancelled) setListing(result) })
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load WooCommerce sync status') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, connection.id, productId])

  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937]/60 bg-gray-100 border dark:border-[#2D3748] border-gray-200 outline-none cursor-not-allowed'
  const labelClass = 'block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5'

  return (
    <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
      <div className="flex items-center gap-2">
        <Plug size={13} className="dark:text-gray-400 text-gray-500" />
        <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">WooCommerce</h3>
        <span className="text-[10px] dark:text-gray-500 text-gray-400">· {connection.store_url}</span>
      </div>

      {loading && (
        <p className="text-[11px] dark:text-gray-500 text-gray-400">Loading sync status…</p>
      )}

      {!loading && error && (
        <p className="text-[11px] dark:text-red-400 text-red-600">{error}</p>
      )}

      {!loading && !error && !listing && (
        <p className="text-[11px] dark:text-gray-500 text-gray-400">
          Not linked to this WooCommerce store yet — import it from WooCommerce or push it from CommerceOS to start syncing.
        </p>
      )}

      {!loading && !error && listing && (
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className={labelClass}>WooCommerce product ID</label>
            <input disabled readOnly value={listing.external_id} className={clsx(inputClass, 'dark:text-gray-300 text-gray-700')} />
          </div>
          <div>
            <label className={labelClass}>Sync status</label>
            <input
              disabled
              readOnly
              value={WOO_SYNC_STATUS_LABEL[listing.sync_status]}
              className={clsx(inputClass, WOO_SYNC_STATUS_STYLE[listing.sync_status])}
            />
          </div>
          <div>
            <label className={labelClass}>Last synced</label>
            <input
              disabled
              readOnly
              value={listing.last_synced_at ? new Date(listing.last_synced_at).toLocaleString() : '—'}
              className={clsx(inputClass, 'dark:text-gray-300 text-gray-700')}
            />
          </div>
        </div>
      )}

      {/* woo_tax_class_slug is set once by import and never cleared, so it
          persists independently of listing (e.g. after "Delete imported
          products" removes the listing but not this SKU) -- shown whenever
          this SKU actually came from Woo, not gated on listing being present. */}
      {sku?.woo_tax_class_slug != null && (
        <div className="grid grid-cols-2 gap-3 pt-3 border-t dark:border-[#2D3748] border-gray-200">
          <div>
            <label htmlFor="product-woo-tax-class" className={labelClass}>Tax class (Woo)</label>
            <input
              id="product-woo-tax-class"
              disabled
              readOnly
              value={sku.woo_tax_class_slug || 'Standard'}
              className={clsx(inputClass, 'dark:text-gray-300 text-gray-700')}
            />
          </div>
          <div>
            <label htmlFor="product-woo-tax-status" className={labelClass}>Tax status (Woo)</label>
            <input
              id="product-woo-tax-status"
              disabled
              readOnly
              value={sku.woo_tax_status || '—'}
              className={clsx(inputClass, 'dark:text-gray-300 text-gray-700')}
            />
          </div>
        </div>
      )}
    </div>
  )
}

// ── Media section ─────────────────────────────────────────────────────────────

const ACCEPTED_IMAGE_TYPES = 'image/jpeg,image/png,image/webp,image/gif'

function MediaSection({ productId }: { productId: string }) {
  const { token } = useAuth()
  const { media, loading, error, upload, update, remove } = useProductMedia(token ?? '', productId)
  const [uploading, setUploading] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [titleDraft, setTitleDraft] = useState('')
  const [titleSaving, setTitleSaving] = useState(false)
  const [titleError, setTitleError] = useState('')
  const [dragging, setDragging] = useState(false)

  const featured = media[0] ?? null
  const rest = media.slice(1)

  // Uploaded sequentially, not in parallel -- each upload's success handler
  // appends to `media` via a functional setState update, but running them
  // one at a time keeps upload order == drop/selection order and avoids
  // firing a burst of simultaneous multipart requests for a large batch.
  async function handleFiles(files: FileList | Iterable<File> | null) {
    if (!files) return
    const list = Array.from(files)
    if (list.length === 0) return
    setUploading(true)
    setUploadError(null)
    try {
      for (const file of list) {
        await upload(file)
      }
    } catch (e) {
      setUploadError(e instanceof Error ? e.message : 'Upload failed')
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  function handleDrop(e: React.DragEvent<HTMLDivElement>) {
    e.preventDefault()
    setDragging(false)
    handleFiles(e.dataTransfer.files)
  }

  function openTitleEditor(m: Media) {
    setEditingId(m.id)
    setTitleDraft(m.alt_text)
    setTitleError('')
  }

  async function handleSaveTitle() {
    if (!editingId) return
    setTitleSaving(true)
    setTitleError('')
    try {
      await update(editingId, titleDraft)
      setEditingId(null)
    } catch (e) {
      setTitleError(e instanceof Error ? e.message : 'Failed to update title')
    } finally {
      setTitleSaving(false)
    }
  }

  return (
    <div className="space-y-3">
      <input
        ref={fileInputRef}
        type="file"
        accept={ACCEPTED_IMAGE_TYPES}
        multiple
        className="hidden"
        onChange={e => handleFiles(e.target.files)}
      />
      <div
        data-testid="media-dropzone"
        onDragOver={e => { e.preventDefault(); setDragging(true) }}
        onDragLeave={() => setDragging(false)}
        onDrop={handleDrop}
        className={clsx(
          'grid grid-cols-2 gap-3 rounded-xl transition-colors',
          dragging && 'ring-2 ring-blue-500 ring-offset-2 ring-offset-transparent',
        )}
      >
        <div className="group relative aspect-square rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden flex items-center justify-center text-[11px] dark:text-gray-600 text-gray-400">
          {featured ? (
            <>
              <button
                type="button"
                onClick={() => openTitleEditor(featured)}
                aria-label="Edit image title"
                className="w-full h-full cursor-pointer"
              >
                <img src={featured.url} alt={featured.alt_text} className="w-full h-full object-cover" />
              </button>
              <button
                type="button"
                onClick={() => remove(featured.id)}
                aria-label="Remove image"
                className="absolute top-1.5 right-1.5 p-1 rounded-full bg-black/60 text-white opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <X size={12} />
              </button>
            </>
          ) : (
            <span>Featured image</span>
          )}
        </div>
        <div className="grid grid-cols-2 gap-2 content-start">
          {rest.map(m => (
            <div key={m.id} className="group relative aspect-square rounded-lg border dark:border-[#2D3748] border-gray-200 overflow-hidden">
              <button
                type="button"
                onClick={() => openTitleEditor(m)}
                aria-label="Edit image title"
                className="w-full h-full cursor-pointer"
              >
                <img src={m.url} alt={m.alt_text} className="w-full h-full object-cover" />
              </button>
              <button
                type="button"
                onClick={() => remove(m.id)}
                aria-label="Remove image"
                className="absolute top-1 right-1 p-0.5 rounded-full bg-black/60 text-white opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <X size={10} />
              </button>
            </div>
          ))}
          <button
            type="button"
            disabled={uploading}
            onClick={() => fileInputRef.current?.click()}
            className="aspect-square rounded-lg border-2 border-dashed dark:border-[#2D3748] border-gray-200 flex items-center justify-center dark:text-gray-600 text-gray-400 hover:dark:text-gray-400 hover:text-gray-500 disabled:opacity-50"
          >
            <ImagePlus size={16} />
          </button>
        </div>
      </div>
      {editingId && (
        <div className="p-2.5 rounded-lg border dark:border-[#2D3748] border-gray-200 space-y-2">
          <label htmlFor="media-title" className="block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">
            Title
          </label>
          <input
            id="media-title"
            type="text"
            value={titleDraft}
            onChange={e => setTitleDraft(e.target.value)}
            placeholder="Describe this image"
            className="w-full px-2.5 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none"
          />
          {titleError && <p className="text-[10px] text-red-500">{titleError}</p>}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setEditingId(null)}
              className="text-[10px] dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors"
            >
              Cancel
            </button>
            <button
              type="button"
              disabled={titleSaving}
              onClick={handleSaveTitle}
              className="text-[10px] font-medium px-2.5 py-1 rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              {titleSaving ? 'Saving…' : 'Save title'}
            </button>
          </div>
        </div>
      )}
      {loading && <p className="text-[11px] dark:text-gray-500 text-gray-400">Loading…</p>}
      {(uploadError || error) && (
        <p className="text-[11px] text-red-500">{uploadError ?? error}</p>
      )}
    </div>
  )
}

// ── Variants + SKUs section (only shown when the product has variants) ───────

interface VariantsSectionProps {
  detail: ReturnType<typeof useProductDetail>
  productId: string
}

function VariantsSection({ detail, productId }: VariantsSectionProps) {
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
          className="text-[10px] font-medium px-2 py-1 rounded-lg dark:bg-[#1F2937] bg-gray-100 dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-200 transition-colors"
        >
          {adding ? 'Cancel' : '+ Add Variant'}
        </button>
      </div>

      {detail.loading && <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
      {detail.error && <p className="text-xs text-red-400">{detail.error}</p>}

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
        // "Add Variant" button redirecting home.
        <div className="p-3 rounded-lg border dark:border-[#2D3748] border-gray-200 space-y-2">
          <input
            type="text" required placeholder="Variant name (e.g. Large)" value={variantName} onChange={e => setVariantName(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); handleAddVariant() } }}
            className="w-full px-2.5 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none"
          />
          {addError && <p className="text-xs text-red-400">{addError}</p>}
          <button type="button" onClick={handleAddVariant} disabled={submitting} className="w-full py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors">
            {submitting ? 'Adding…' : 'Add Variant'}
          </button>
          <p className="text-[10px] dark:text-gray-500 text-gray-400">Add a SKU (price/barcode) to the variant after creating it.</p>
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

function CaseOfSKUPicker({ excludeSKUID, filterSku, onSelect, onClose }: CaseOfSKUPickerProps) {
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
    <div className="rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#0B1220] bg-white p-3 space-y-2">
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
          <input
            autoFocus value={query} onChange={e => setQuery(e.target.value)}
            placeholder="Search products by name…"
            className="w-full pl-7 pr-2 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <button type="button" onClick={onClose} title="Close">
          <X size={14} className="dark:text-gray-500 text-gray-400" />
        </button>
      </div>
      <div className="max-h-48 overflow-auto space-y-1">
        {products.length === 0 && <p className="text-[10px] dark:text-gray-500 text-gray-400 px-1">No matching products</p>}
        {products.map(p => (
          <div key={p.id}>
            <button
              type="button"
              onClick={() => toggleProduct(p.id)}
              className="w-full text-left px-2 py-1.5 text-xs rounded dark:hover:bg-[#1F2937] hover:bg-gray-50 dark:text-gray-200 text-gray-800"
            >
              {p.name}
            </button>
            {expandedProductId === p.id && (
              <div className="pl-4 space-y-0.5">
                {loadingSkus && <p className="text-[10px] dark:text-gray-500 text-gray-400">Loading…</p>}
                {!loadingSkus && skus.length === 0 && <p className="text-[10px] dark:text-gray-500 text-gray-400">No other SKUs on this product</p>}
                {!loadingSkus && skus.map(s => (
                  <button
                    key={s.id}
                    type="button"
                    onClick={() => onSelect(s, `${p.name} — ${s.sku_code}`)}
                    className="w-full flex items-center justify-between text-left px-2 py-1 text-[10px] rounded dark:hover:bg-[#1F2937] hover:bg-gray-50"
                  >
                    <span className="font-mono dark:text-gray-300 text-gray-600">{s.sku_code}</span>
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
function StockBadge({ skuId }: { skuId: string }) {
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
      className="flex items-center gap-1 text-[10px] dark:text-gray-400 text-gray-500 hover:text-blue-400 shrink-0"
    >
      <Boxes size={11} />
      {level ? `${level.total} in stock` : '…'}
    </button>
  )
}

interface VariantRowProps {
  variant: VariantWithSKUs
  detail: ReturnType<typeof useProductDetail>
  productMedia: Media[]
}

function VariantRow({ variant, detail, productMedia }: VariantRowProps) {
  const { token } = useAuth()
  const { settings: tenantSettings } = useTenantSettings(token ?? '')
  const [addingSKU, setAddingSKU] = useState(false)
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
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  // See ProductEditPage's identical currency-defaulting comment -- same
  // "wait for tenantSettings to actually arrive" reasoning.
  const [currencyDefaulted, setCurrencyDefaulted] = useState(false)
  if (!currencyDefaulted && tenantSettings) {
    if (tenantSettings.default_currency) setCurrency(tenantSettings.default_currency)
    setCurrencyDefaulted(true)
  }

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
        status,
      })
      setAddingSKU(false)
      setSkuCode(''); setBarcode(''); setPrice(''); setSalePrice(''); setSaleEndDate('')
      setCurrency(tenantSettings?.default_currency || 'USD'); setStatus('active')
      setIsVirtual(false); setWeightGrams(''); setLengthMM(''); setWidthMM(''); setHeightMM('')
      setUnitType('unit'); setPortionedFromSKUID(null); setPortionedFromLabel('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add SKU')
    } finally {
      setSubmitting(false)
    }
  }

  const sku = variant.skus?.[0]
  const [bulkSaving, setBulkSaving] = useState(false)
  const [bulkError, setBulkError] = useState('')
  const [existingPortionPickerOpen, setExistingPortionPickerOpen] = useState(false)

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
        status: sku.status,
      })
      setExistingPortionPickerOpen(false)
    } catch (err) {
      setBulkError(err instanceof Error ? err.message : 'Failed to update weight pool link')
    } finally {
      setBulkSaving(false)
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
    <Package size={14} className="dark:text-gray-500 text-gray-400" />
  )

  return (
    <div className="p-3 rounded-lg border dark:border-[#2D3748] border-gray-200">
      <div className="flex items-center gap-3">
        {productMedia.length > 0 ? (
          <button
            type="button"
            aria-label={`${variant.name} image`}
            onClick={() => setPickerOpen(v => !v)}
            className="w-10 h-10 rounded-lg dark:bg-[#1F2937] bg-gray-100 flex items-center justify-center shrink-0 overflow-hidden"
          >
            {thumbnailContent}
          </button>
        ) : (
          <div className="w-10 h-10 rounded-lg dark:bg-[#1F2937] bg-gray-100 flex items-center justify-center shrink-0 overflow-hidden">
            {thumbnailContent}
          </div>
        )}
        <div className="flex-1 min-w-0">
          <p className="text-xs font-medium dark:text-gray-200 text-gray-800 truncate">{variant.name}</p>
          <p className="text-[10px] font-mono dark:text-gray-500 text-gray-400 truncate">
            {sku ? sku.sku_code : 'No SKU yet'}
          </p>
        </div>
        {sku && sku.sale_price_minor_units != null ? (
          <span className="flex items-center gap-1.5 shrink-0">
            <span className="text-[10px] dark:text-gray-500 text-gray-400 line-through">
              {formatPrice(sku.price_minor_units, sku.currency_code)}
            </span>
            <span className="text-xs font-medium text-green-400">
              {formatPrice(sku.sale_price_minor_units, sku.currency_code)}
            </span>
            {sku.sale_end_date && (
              <span className="text-[9px] dark:text-gray-500 text-gray-400">until {sku.sale_end_date}</span>
            )}
          </span>
        ) : sku && (
          <span className="text-xs font-medium dark:text-gray-200 text-gray-800 shrink-0">
            {formatPrice(sku.price_minor_units, sku.currency_code)}
          </span>
        )}
        {sku && (
          <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium capitalize shrink-0', SKU_STATUS_STYLES[sku.status])}>
            {sku.status}
          </span>
        )}
        {sku && <StockBadge skuId={sku.id} />}
        <button
          type="button"
          onClick={() => detail.removeVariant(variant.id)}
          className="text-[10px] dark:text-gray-500 text-gray-400 hover:text-red-400 transition-colors shrink-0"
        >
          Remove
        </button>
      </div>

      {pickerOpen && (
        <div className="flex flex-wrap gap-1.5 mt-2 p-2 rounded-lg dark:bg-[#1F2937]/60 bg-gray-50 border dark:border-[#2D3748] border-gray-200">
          <button
            type="button"
            onClick={() => handleSelectImage(null)}
            aria-label="No image"
            className={clsx(
              'w-8 h-8 rounded-md border flex items-center justify-center text-[10px] dark:text-gray-500 text-gray-400 shrink-0',
              !variant.featured_media_id ? 'border-blue-500' : 'dark:border-[#2D3748] border-gray-200',
            )}
          >
            <X size={12} />
          </button>
          {productMedia.map(m => (
            <button
              key={m.id}
              type="button"
              onClick={() => handleSelectImage(m.id)}
              aria-label={m.alt_text || 'Product image'}
              className={clsx(
                'w-8 h-8 rounded-md overflow-hidden border shrink-0',
                m.id === variant.featured_media_id ? 'border-blue-500' : 'dark:border-[#2D3748] border-gray-200',
              )}
            >
              <img src={m.url} alt={m.alt_text} className="w-full h-full object-cover" />
            </button>
          ))}
        </div>
      )}

      {imageError && <p className="text-[10px] text-red-500 mt-1.5">{imageError}</p>}

      {sku && (
        <div className="mt-1.5 space-y-1">
          <div className="flex items-center justify-between">
            {sku.unit_type === 'kilo' ? (
              <span className="text-[10px] dark:text-gray-400 text-gray-500 flex items-center gap-1.5">
                Weight pool
                <button type="button" disabled={bulkSaving} onClick={() => saveBulkPool({ unitType: 'unit', portionedFromSkuId: null })} className="hover:underline disabled:opacity-50">
                  Turn off
                </button>
              </span>
            ) : sku.portioned_from_sku_id ? (
              <span className="text-[10px] dark:text-gray-400 text-gray-500 flex items-center gap-1.5">
                Portioned from <span className="font-medium">{sku.portioned_from_sku_id}</span>
                <button type="button" disabled={bulkSaving} onClick={() => saveBulkPool({ unitType: sku.unit_type ?? 'unit', portionedFromSkuId: null })} className="hover:underline disabled:opacity-50">
                  Unlink
                </button>
              </span>
            ) : existingPortionPickerOpen ? null : (
              <div className="flex items-center gap-2">
                <button type="button" disabled={bulkSaving} onClick={() => saveBulkPool({ unitType: 'kilo', portionedFromSkuId: null })} className="text-[10px] text-blue-400 hover:text-blue-300 transition-colors disabled:opacity-50">
                  Make weight pool
                </button>
                <button type="button" onClick={() => setExistingPortionPickerOpen(true)} className="text-[10px] text-blue-400 hover:text-blue-300 transition-colors">
                  Link to weight pool
                </button>
              </div>
            )}
            <button type="button" onClick={() => detail.removeSKU(sku.id)} className="text-[10px] dark:text-gray-500 text-gray-400 hover:text-red-400 transition-colors">
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
          {bulkError && <p className="text-[10px] text-red-400">{bulkError}</p>}
        </div>
      )}

      {!sku && !addingSKU && (
        <button type="button" onClick={() => setAddingSKU(true)} className="text-[10px] text-blue-400 hover:text-blue-300 transition-colors mt-2">
          + Add SKU
        </button>
      )}

      {addingSKU && (
        // Plain div, not <form> — same nested-<form>-inside-the-page's-outer-
        // form bug as the Add Variant form above (see comment there).
        <div className="mt-2 space-y-1.5">
          <div className="grid grid-cols-2 gap-1.5">
            <input type="text" required placeholder="SKU code" value={skuCode} onChange={e => setSkuCode(e.target.value)}
              className="px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none" />
            <input type="text" placeholder="Barcode" value={barcode} onChange={e => setBarcode(e.target.value)}
              className="px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none" />
            <input type="number" step="0.01" min="0" required placeholder="Price" value={price} onChange={e => setPrice(e.target.value)}
              className="px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none" />
            <input type="number" step="0.01" min="0" placeholder="Sale price (optional)" value={salePrice} onChange={e => setSalePrice(e.target.value)}
              className="px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none" />
            <input type="date" title="Sale ends (optional)" value={saleEndDate} onChange={e => setSaleEndDate(e.target.value)}
              className="px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none" />
            <select value={currency} onChange={e => setCurrency(e.target.value)}
              className="px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none">
              {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
            </select>
          </div>
          <select value={status} onChange={e => setStatus(e.target.value as SKUStatus)}
            className="w-full px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none">
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
          </select>
          <label className="flex items-center gap-1.5 text-[10px] dark:text-gray-400 text-gray-500">
            <input type="checkbox" checked={isVirtual} onChange={e => setIsVirtual(e.target.checked)} />
            Virtual (no shipping)
          </label>
          {!isVirtual && (
            <div className="grid grid-cols-2 gap-1.5">
              <input type="number" step="1" min="0" placeholder="Weight (g)" value={weightGrams} onChange={e => setWeightGrams(e.target.value)}
                className="px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none" />
              <input type="number" step="1" min="0" placeholder="Length (mm)" value={lengthMM} onChange={e => setLengthMM(e.target.value)}
                className="px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none" />
              <input type="number" step="1" min="0" placeholder="Width (mm)" value={widthMM} onChange={e => setWidthMM(e.target.value)}
                className="px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none" />
              <input type="number" step="1" min="0" placeholder="Height (mm)" value={heightMM} onChange={e => setHeightMM(e.target.value)}
                className="px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none" />
            </div>
          )}
          <label className="flex items-center gap-1.5 text-[10px] dark:text-gray-400 text-gray-500">
            Item type
            <select
              value={unitType}
              onChange={e => {
                const next = e.target.value as SKUUnitType
                setUnitType(next)
                if (next === 'kilo') { setPortionedFromSKUID(null); setPortionedFromLabel('') }
              }}
              className="px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none"
            >
              <option value="unit">Unit</option>
              <option value="box">Box</option>
              <option value="kilo">Kilo (weight pool)</option>
            </select>
          </label>
          {unitType !== 'kilo' && (
            portionedFromSKUID ? (
              <div className="flex items-center gap-1.5 text-[10px] dark:text-gray-400 text-gray-500">
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
              <button type="button" onClick={() => setPortionPickerOpen(true)} className="text-[10px] text-blue-400 hover:text-blue-300 transition-colors">
                + Link to a weight pool
              </button>
            )
          )}
          {error && <p className="text-[10px] text-red-400">{error}</p>}
          <div className="flex gap-1.5">
            <button type="button" onClick={handleAddSKU} disabled={submitting} className="flex-1 py-1 rounded bg-blue-600 text-white text-[10px] font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors">
              {submitting ? 'Adding…' : 'Add'}
            </button>
            <button type="button" onClick={() => setAddingSKU(false)} className="px-2 py-1 rounded dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 transition-colors">
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
