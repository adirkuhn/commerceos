import { useState, useEffect } from 'react'
import {
  Plus, Search, Package, Trash2, Edit3, ArrowLeft, Plug,
} from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useProducts } from '../pms/hooks/useProducts'
import { useCategories } from '../pms/hooks/useCategories'
import { useProductDetail } from '../pms/hooks/useProductDetail'
import * as pmsApi from '../pms/api'
import * as integrationsApi from '../integrations/api'
import type { Connection } from '../integrations/types'
import type { Product, ProductStatus, SKUStatus, VariantWithSKUs } from '../pms/types'

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

function formatPrice(minorUnits: number, currency: string): string {
  return `${minorUnits < 0 ? '-' : ''}${currency} ${(Math.abs(minorUnits) / 100).toFixed(2)}`
}

function parsePriceToMinorUnits(price: string): number {
  return Math.round(parseFloat(price || '0') * 100)
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
  const { products, loading, error, create: createProduct, update: updateProduct, remove: removeProduct } = useProducts(token ?? '')
  const { categories, create: createCategory } = useCategories(token ?? '')

  const [search, setSearch] = useState('')
  const [editingId, setEditingId] = useState<string | null | 'new'>(null)

  const categoryName = (id?: string | null) => categories.find(c => c.id === id)?.name ?? '—'

  const filtered = products.filter(p =>
    p.name.toLowerCase().includes(search.toLowerCase()) ||
    p.slug.toLowerCase().includes(search.toLowerCase()),
  )

  if (editingId !== null) {
    return (
      <ProductEditPage
        productId={editingId === 'new' ? null : editingId}
        categories={categories}
        onCreateCategory={createCategory}
        onCreate={createProduct}
        onUpdate={updateProduct}
        onClose={() => setEditingId(null)}
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
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search by name or slug..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white dark:border-[#2D3748] border-gray-200 border dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <button
            onClick={() => setEditingId('new')}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 transition-colors font-medium"
          >
            <Plus size={13} /> New Product
          </button>
        </div>
      </div>

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">{filtered.length} products</span>
        <span className="text-xs dark:text-gray-500 text-gray-400">·</span>
        <span className="text-xs dark:text-gray-500 text-gray-400">
          <span className="text-green-400 font-medium">{products.filter(p => p.status === 'active').length}</span> active
        </span>
        {error && <span className="text-xs text-red-400 ml-auto">{error}</span>}
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Product</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Category</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 w-8" />
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={5} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={5} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No products yet</td></tr>
            )}
            {filtered.map(p => (
              <tr
                key={p.id}
                onClick={() => setEditingId(p.id)}
                className="group hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-[#2D3748] bg-gray-200 flex items-center justify-center">
                    <Package size={12} className="dark:text-gray-500 text-gray-400" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[220px]">
                  <p className="font-medium dark:text-gray-100 text-gray-900 truncate">{p.name}</p>
                  <p className="text-[10px] font-mono dark:text-gray-500 text-gray-400 truncate">{p.slug}</p>
                </td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{categoryName(p.category_id)}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium capitalize', STATUS_STYLES[p.status])}>
                    {p.status}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={e => { e.stopPropagation(); setEditingId(p.id) }}
                      className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors"
                      title="Edit"
                    >
                      <Edit3 size={12} className="dark:text-gray-400 text-gray-500" />
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

type EditorTab = 'general' | 'woocommerce'

interface ProductEditorProps {
  productId: string | null
  categories: { id: string; name: string }[]
  onCreateCategory: (payload: { name: string }) => Promise<{ id: string; name: string }>
  onCreate: (payload: { name: string; description?: string; category_id?: string | null; status: ProductStatus; has_variants?: boolean }) => Promise<Product>
  onUpdate: (id: string, payload: { name: string; description?: string; category_id?: string | null; status: ProductStatus; has_variants?: boolean }) => Promise<Product>
  onClose: () => void
}

function ProductEditPage({ productId, categories, onCreateCategory, onCreate, onUpdate, onClose }: ProductEditorProps) {
  const { token } = useAuth()
  const [savedProductId, setSavedProductId] = useState<string | null>(productId)
  const detail = useProductDetail(token ?? '', savedProductId)

  const existing = detail.detail?.product
  const [name, setName] = useState(existing?.name ?? '')
  const [description, setDescription] = useState(existing?.description ?? '')
  const [categoryId, setCategoryId] = useState<string>(existing?.category_id ?? '')
  const [status, setStatus] = useState<ProductStatus>(existing?.status ?? 'draft')
  const [newCategoryName, setNewCategoryName] = useState('')
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')
  const [activeTab, setActiveTab] = useState<EditorTab>('general')

  // Integration tabs only ever appear for a provider that's actually
  // connected — an unconnected/disabled integration gets no tab at all,
  // rather than a tab that just tells you to go connect it.
  const [wooConnection, setWooConnection] = useState<Connection | null>(null)
  useEffect(() => {
    if (!token) return
    integrationsApi.listConnections(token)
      .then(conns => setWooConnection(conns.find(c => c.provider === 'woocommerce' && c.status === 'connected') ?? null))
      .catch(() => setWooConnection(null))
  }, [token])

  // Whether to show the full named-variants manager vs. the flat single-SKU
  // form. For a brand-new product this is a user choice made before saving;
  // for an existing product it's the server's own has_variants field —
  // source of truth lives in the DB, not inferred from variant count.
  const [variantsMode, setVariantsMode] = useState(false)

  // Simple-mode SKU fields (only relevant when variantsMode is false).
  const [skuCode, setSkuCode] = useState('')
  const [barcode, setBarcode] = useState('')
  const [price, setPrice] = useState('')
  const [currency, setCurrency] = useState('USD')
  const [skuStatus, setSkuStatus] = useState<SKUStatus>('active')
  const [isVirtual, setIsVirtual] = useState(false)
  const [weightGrams, setWeightGrams] = useState('')
  const [lengthMM, setLengthMM] = useState('')
  const [widthMM, setWidthMM] = useState('')
  const [heightMM, setHeightMM] = useState('')

  // Sync local form state once the product detail loads (edit mode) — runs once.
  const [hydrated, setHydrated] = useState(false)
  if (!hydrated && existing) {
    setName(existing.name)
    setDescription(existing.description)
    setCategoryId(existing.category_id ?? '')
    setStatus(existing.status)
    setVariantsMode(existing.has_variants)
    const variants = detail.detail?.variants ?? []
    if (variants.length === 1) {
      const sku = variants[0].skus?.[0]
      if (sku) {
        setSkuCode(sku.sku_code)
        setBarcode(sku.barcode ?? '')
        setPrice(formatMinorUnitsAsInput(sku.price_minor_units))
        setCurrency(sku.currency_code)
        setSkuStatus(sku.status)
        setIsVirtual(sku.is_virtual)
        setWeightGrams(sku.weight_grams != null ? String(sku.weight_grams) : '')
        setLengthMM(sku.length_mm != null ? String(sku.length_mm) : '')
        setWidthMM(sku.width_mm != null ? String(sku.width_mm) : '')
        setHeightMM(sku.height_mm != null ? String(sku.height_mm) : '')
      }
    }
    setHydrated(true)
  }

  const singleVariant: VariantWithSKUs | null =
    !variantsMode && detail.detail?.variants.length === 1 ? detail.detail.variants[0] : null

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      const payload = { name, description, category_id: categoryId || null, status, has_variants: variantsMode }
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
              currency_code: currency,
              is_virtual: isVirtual,
              ...dimensions,
              status: skuStatus,
            })
          }
        }
        return
      }

      await onUpdate(savedProductId, payload)
      if (!variantsMode) {
        if (singleVariant) {
          const currentSku = singleVariant.skus?.[0]
          if (currentSku) {
            await detail.updateSKU(currentSku.id, {
              barcode: barcode || undefined,
              price_minor_units: parsePriceToMinorUnits(price),
              currency_code: currency,
              is_virtual: isVirtual,
              ...dimensions,
              status: skuStatus,
            })
          } else if (skuCode) {
            await detail.addSKU(singleVariant.id, {
              sku_code: skuCode,
              barcode: barcode || undefined,
              price_minor_units: parsePriceToMinorUnits(price),
              currency_code: currency,
              is_virtual: isVirtual,
              ...dimensions,
              status: skuStatus,
            })
          }
        } else if (skuCode) {
          const variant = await pmsApi.createVariant(token ?? '', savedProductId, { name })
          await pmsApi.createSKU(token ?? '', variant.id, {
            sku_code: skuCode,
            barcode: barcode || undefined,
            price_minor_units: parsePriceToMinorUnits(price),
            currency_code: currency,
            is_virtual: isVirtual,
            ...dimensions,
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
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className={clsx('max-w-5xl mx-auto p-6', activeTab !== 'general' && 'hidden')}>
          {renderGeneralTab()}
        </div>
        {activeTab === 'woocommerce' && savedProductId && wooConnection && (
          <div className="max-w-5xl mx-auto p-6">
            <ProductWooCommerceTab productId={savedProductId} connection={wooConnection} />
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
              <div>
                <label htmlFor="product-description" className={labelClass}>Description</label>
                <textarea
                  id="product-description"
                  value={description} onChange={e => setDescription(e.target.value)} rows={3}
                  className={inputClass}
                />
              </div>
            </FormCard>

            <FormCard title="Media" description="Coming soon — image upload isn't wired up yet.">
              <div className="grid grid-cols-2 gap-3">
                <div className="aspect-square rounded-xl border-2 border-dashed dark:border-[#2D3748] border-gray-200 flex items-center justify-center text-[11px] dark:text-gray-600 text-gray-400">
                  Featured image
                </div>
                <div className="grid grid-cols-2 gap-2 content-start">
                  {[1, 2, 3].map(i => (
                    <div key={i} className="aspect-square rounded-lg border-2 border-dashed dark:border-[#2D3748] border-gray-200 flex items-center justify-center text-[10px] dark:text-gray-600 text-gray-400">
                      —
                    </div>
                  ))}
                  <button
                    type="button" disabled
                    className="aspect-square rounded-lg border-2 border-dashed dark:border-[#2D3748] border-gray-200 flex items-center justify-center text-[10px] dark:text-gray-600 text-gray-400 cursor-not-allowed"
                  >
                    + Upload
                  </button>
                </div>
              </div>
            </FormCard>

            <FormCard title="Variants">
              {variantsMode ? (
                <>
                  {savedProductId ? (
                    <VariantsSection detail={detail} />
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
                <button
                  type="button"
                  onClick={() => setVariantsMode(true)}
                  className="text-xs font-medium text-blue-400 hover:text-blue-300 transition-colors"
                >
                  + Add variants like size or color
                </button>
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
                  </div>
                  <div>
                    <label htmlFor="product-currency" className={labelClass}>Currency</label>
                    <select id="product-currency" value={currency} onChange={e => setCurrency(e.target.value)} className={inputClass}>
                      {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
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

            {formError && <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-3 py-2">{formError}</p>}
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
                  {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
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
              </div>

              <div className="pt-4">
                <label htmlFor="product-vendor" className={labelClass}>Vendor</label>
                <input id="product-vendor" disabled placeholder="Coming soon" className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
              </div>

              <div className="pt-4">
                <label htmlFor="product-collections" className={labelClass}>Collections</label>
                <input id="product-collections" disabled placeholder="Coming soon" className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
              </div>

              <div className="pt-4">
                <label htmlFor="product-tags" className={labelClass}>Tags</label>
                <input id="product-tags" disabled placeholder="Coming soon" className={clsx(inputClass, 'cursor-not-allowed opacity-60')} />
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
// Per-channel values for this product on the connected WooCommerce store.
// Shell only for now — fields are disabled placeholders, nothing here is
// persisted yet (no backend concept of a per-integration product mapping
// exists). Wiring this up for real (external product id, sync status, price
// override) is a separate, larger change: new domain entity, migration,
// OpenAPI contract, sync use cases.

function ProductWooCommerceTab({ productId, connection }: { productId: string; connection: Connection }) {
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937]/60 bg-gray-100 border dark:border-[#2D3748] border-gray-200 dark:text-gray-500 text-gray-400 outline-none cursor-not-allowed'
  const labelClass = 'block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5'

  return (
    <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
      <div className="flex items-center gap-2">
        <Plug size={13} className="dark:text-gray-400 text-gray-500" />
        <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">WooCommerce</h3>
        <span className="text-[10px] dark:text-gray-500 text-gray-400">· {connection.store_url}</span>
      </div>
      <p className="text-[11px] dark:text-gray-500 text-gray-400">
        Per-channel sync isn't wired up yet — these fields are a preview of what's coming for product {productId}.
      </p>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className={labelClass}>WooCommerce product ID</label>
          <input disabled placeholder="Not linked yet" className={inputClass} />
        </div>
        <div>
          <label className={labelClass}>Sync status</label>
          <input disabled placeholder="Not synced" className={inputClass} />
        </div>
        <div>
          <label className={labelClass}>Last synced</label>
          <input disabled placeholder="—" className={inputClass} />
        </div>
        <div>
          <label className={labelClass}>Price override</label>
          <input disabled placeholder="Uses base price" className={inputClass} />
        </div>
      </div>
    </div>
  )
}

// ── Variants + SKUs section (only shown when the product has variants) ───────

interface VariantsSectionProps {
  detail: ReturnType<typeof useProductDetail>
}

function VariantsSection({ detail }: VariantsSectionProps) {
  const [adding, setAdding] = useState(false)
  const [variantName, setVariantName] = useState('')
  const [addError, setAddError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const resetForm = () => {
    setAdding(false)
    setVariantName('')
    setAddError('')
  }

  const handleAddVariant = async (e: React.FormEvent) => {
    e.preventDefault()
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
          <VariantRow key={v.id} variant={v} detail={detail} />
        ))}
      </div>

      {adding && (
        <form onSubmit={handleAddVariant} className="p-3 rounded-lg border dark:border-[#2D3748] border-gray-200 space-y-2">
          <input
            type="text" required placeholder="Variant name (e.g. Large)" value={variantName} onChange={e => setVariantName(e.target.value)}
            className="w-full px-2.5 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none"
          />
          {addError && <p className="text-xs text-red-400">{addError}</p>}
          <button type="submit" disabled={submitting} className="w-full py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors">
            {submitting ? 'Adding…' : 'Add Variant'}
          </button>
          <p className="text-[10px] dark:text-gray-500 text-gray-400">Add a SKU (price/barcode) to the variant after creating it.</p>
        </form>
      )}
    </div>
  )
}

interface VariantRowProps {
  variant: VariantWithSKUs
  detail: ReturnType<typeof useProductDetail>
}

function VariantRow({ variant, detail }: VariantRowProps) {
  const [addingSKU, setAddingSKU] = useState(false)
  const [skuCode, setSkuCode] = useState('')
  const [barcode, setBarcode] = useState('')
  const [price, setPrice] = useState('')
  const [currency, setCurrency] = useState('USD')
  const [status, setStatus] = useState<SKUStatus>('active')
  const [isVirtual, setIsVirtual] = useState(false)
  const [weightGrams, setWeightGrams] = useState('')
  const [lengthMM, setLengthMM] = useState('')
  const [widthMM, setWidthMM] = useState('')
  const [heightMM, setHeightMM] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleAddSKU = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      await detail.addSKU(variant.id, {
        sku_code: skuCode,
        barcode: barcode || undefined,
        price_minor_units: parsePriceToMinorUnits(price),
        currency_code: currency,
        is_virtual: isVirtual,
        weight_grams: isVirtual ? undefined : parseIntOrUndefined(weightGrams),
        length_mm: isVirtual ? undefined : parseIntOrUndefined(lengthMM),
        width_mm: isVirtual ? undefined : parseIntOrUndefined(widthMM),
        height_mm: isVirtual ? undefined : parseIntOrUndefined(heightMM),
        status,
      })
      setAddingSKU(false)
      setSkuCode(''); setBarcode(''); setPrice(''); setCurrency('USD'); setStatus('active')
      setIsVirtual(false); setWeightGrams(''); setLengthMM(''); setWidthMM(''); setHeightMM('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add SKU')
    } finally {
      setSubmitting(false)
    }
  }

  const sku = variant.skus?.[0]

  return (
    <div className="p-3 rounded-lg border dark:border-[#2D3748] border-gray-200">
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 rounded-lg dark:bg-[#1F2937] bg-gray-100 flex items-center justify-center shrink-0">
          <Package size={14} className="dark:text-gray-500 text-gray-400" />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-xs font-medium dark:text-gray-200 text-gray-800 truncate">{variant.name}</p>
          <p className="text-[10px] font-mono dark:text-gray-500 text-gray-400 truncate">
            {sku ? sku.sku_code : 'No SKU yet'}
          </p>
        </div>
        {sku && (
          <span className="text-xs font-medium dark:text-gray-200 text-gray-800 shrink-0">
            {formatPrice(sku.price_minor_units, sku.currency_code)}
          </span>
        )}
        {sku && (
          <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium capitalize shrink-0', SKU_STATUS_STYLES[sku.status])}>
            {sku.status}
          </span>
        )}
        <button
          type="button"
          onClick={() => detail.removeVariant(variant.id)}
          className="text-[10px] dark:text-gray-500 text-gray-400 hover:text-red-400 transition-colors shrink-0"
        >
          Remove
        </button>
      </div>

      {sku && (
        <div className="flex justify-end mt-1.5">
          <button type="button" onClick={() => detail.removeSKU(sku.id)} className="text-[10px] dark:text-gray-500 text-gray-400 hover:text-red-400 transition-colors">
            Remove SKU
          </button>
        </div>
      )}

      {!sku && !addingSKU && (
        <button type="button" onClick={() => setAddingSKU(true)} className="text-[10px] text-blue-400 hover:text-blue-300 transition-colors mt-2">
          + Add SKU
        </button>
      )}

      {addingSKU && (
        <form onSubmit={handleAddSKU} className="mt-2 space-y-1.5">
          <div className="grid grid-cols-2 gap-1.5">
            <input type="text" required placeholder="SKU code" value={skuCode} onChange={e => setSkuCode(e.target.value)}
              className="px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none" />
            <input type="text" placeholder="Barcode" value={barcode} onChange={e => setBarcode(e.target.value)}
              className="px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none" />
            <input type="number" step="0.01" min="0" required placeholder="Price" value={price} onChange={e => setPrice(e.target.value)}
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
          {error && <p className="text-[10px] text-red-400">{error}</p>}
          <div className="flex gap-1.5">
            <button type="submit" disabled={submitting} className="flex-1 py-1 rounded bg-blue-600 text-white text-[10px] font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors">
              {submitting ? 'Adding…' : 'Add'}
            </button>
            <button type="button" onClick={() => setAddingSKU(false)} className="px-2 py-1 rounded dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 transition-colors">
              Cancel
            </button>
          </div>
        </form>
      )}
    </div>
  )
}
