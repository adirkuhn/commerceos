import { useQueryEnum } from '../routing/useQueryState'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { ProductBolTab } from './ProductBolTab'
import type { Category, Product, ProductStatus } from './types'
import { parsePriceToMinorUnits } from './productFormat'
import { ProductWooCommerceTab } from './ProductWooCommerceTab'
import { MediaSection } from './MediaSection'
import { FormCard } from './FormCard'
import { MarginInsightCard, PriceHistoryCard, ProductHistoryCard } from './ProductPricingCards'
import { useChannelConnections } from './hooks/useChannelConnections'
import { useSkuTaxContext } from './hooks/useSkuTaxContext'
import { useProductEditForm } from './hooks/useProductEditForm'
import { computeTaxPreview } from './skuTax'
import { skuPassThroughPayload } from './skuPayloads'
import { EDITOR_TABS, ProductEditorTabs, type EditorTab } from './ProductEditorTabs'
import { ProductEditHeader } from './ProductEditHeader'
import { ProductDetailsCard } from './ProductDetailsCard'
import { ProductVariantsCard } from './ProductVariantsCard'
import { ProductSidebar } from './ProductSidebar'
import { FlatSkuPricingCard } from './FlatSkuPricingCard'
import { FlatSkuCasePackCard } from './FlatSkuCasePackCard'
import { FlatSkuScaleBarcodeCard } from './FlatSkuScaleBarcodeCard'
import { FlatSkuWeightPoolCard } from './FlatSkuWeightPoolCard'
import { FlatSkuShippingCard } from './FlatSkuShippingCard'

// ── Product Edit Page ────────────────────────────────────────────────────────
//
// Dedicated full page (not a slide-over) for creating/editing a product.
// "General" tab holds the shared data every provider needs (name, price,
// variants...). Additional tabs hold provider-specific values — only
// WooCommerce for now, one tab per connected integration eventually
// (Shopify, Lightspeed, ...), each mapping the same product to that
// channel's own fields without polluting the generic model.
//
// This component only composes: form state lives in useProductEditForm,
// each card is its own component.

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
  const { taxClasses, defaultTaxClass, tenantSettings, rateFor } = useSkuTaxContext()
  const form = useProductEditForm({ token, productId, tenantSettings, onCreate, onUpdate })
  const { savedProductId, detail, fields, setField, sku, setSkuField, patchSku, currentSku, showFlatSKU } = form
  const connections = useChannelConnections(token)
  const [requestedTab, setActiveTab] = useQueryEnum<EditorTab>('tab', EDITOR_TABS, 'general')

  const tabAvailable: Record<EditorTab, boolean> = {
    general: true,
    woocommerce: !!(savedProductId && connections.woo),
    bol: !!(savedProductId && connections.bol),
  }
  const availableTabs = EDITOR_TABS.filter(tab => tabAvailable[tab])
  // ?tab= can name a channel tab this product doesn't have (stale link, or
  // connections still loading) -- show General rather than an empty page.
  const activeTab: EditorTab = availableTabs.includes(requestedTab) ? requestedTab : 'general'

  const taxRateForPreview = rateFor(sku.taxClassId)
  const taxPreview = computeTaxPreview(
    sku.price.trim() === '' ? null : parsePriceToMinorUnits(sku.price), taxRateForPreview, tenantSettings?.prices_include_tax,
  )

  const handleAddCategory = async (name: string) => {
    try {
      const category = await onCreateCategory({ name })
      setField('categoryId', category.id)
      return true
    } catch (err) {
      form.setFormError(err instanceof Error ? err.message : 'Failed to create category')
      return false
    }
  }

  return (
    <div className="flex flex-col h-full">
      <ProductEditHeader
        token={token} savedProductId={savedProductId} name={fields.name} showResync={!!connections.woo}
        saving={form.saving} saveDisabled={activeTab !== 'general'} onClose={onClose}
      />

      <ProductEditorTabs tabs={availableTabs} active={activeTab} onSelect={setActiveTab} />

      {form.formError && (
        // Rendered here (shrink-0, above the scrolling form body) rather
        // than inline in the General tab -- a save error must be visible
        // without the user having to scroll past every FormCard to the
        // bottom of the page to notice the save silently failed.
        <div className="px-6 pt-3 shrink-0">
          <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-3 py-2">{form.formError}</p>
        </div>
      )}

      <div className="flex-1 overflow-y-auto">
        <div className={clsx('max-w-5xl mx-auto p-6', activeTab !== 'general' && 'hidden')}>
          <form id="product-general-form" onSubmit={form.handleSave}>
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
              {/* Main column */}
              <div className="lg:col-span-2 space-y-5">
                <ProductDetailsCard fields={fields} setField={setField} showSlug={!!savedProductId} />

                <FormCard title="Media">
                  {savedProductId ? (
                    <MediaSection productId={savedProductId} />
                  ) : (
                    <p className="text-2xs text-ink-4">Save the product first, then add images.</p>
                  )}
                </FormCard>

                {savedProductId && <ProductHistoryCard productId={savedProductId} />}

                <ProductVariantsCard
                  detail={detail} savedProductId={savedProductId} variantsMode={form.variantsMode}
                  showWooVariationsToggle={!!connections.woo}
                  syncVariantsAsWooVariations={form.syncVariantsAsWooVariations}
                  onSyncVariantsAsWooVariationsChange={form.setSyncVariantsAsWooVariations}
                  enablingVariants={form.enablingVariants} enableVariantsError={form.enableVariantsError}
                  onEnableVariants={form.handleEnableVariants} onDisableVariants={() => form.setVariantsMode(false)}
                />

                {showFlatSKU && (
                  <FlatSkuPricingCard
                    form={sku} setField={setSkuField} currentSkuId={currentSku?.id ?? null}
                    taxClasses={taxClasses} defaultTaxClass={defaultTaxClass} taxPreview={taxPreview}
                  />
                )}

                {showFlatSKU && currentSku && (
                  <MarginInsightCard
                    skuId={currentSku.id} currency={sku.currency}
                    taxRateBasisPoints={taxRateForPreview?.rate_basis_points ?? null}
                    pricesIncludeTax={tenantSettings?.prices_include_tax ?? false}
                  />
                )}

                {showFlatSKU && currentSku && (
                  <PriceHistoryCard skuId={currentSku.id} currency={sku.currency} />
                )}

                {showFlatSKU && currentSku && (
                  <FlatSkuCasePackCard form={sku} setField={setSkuField} patch={patchSku} currentSkuId={currentSku.id} />
                )}

                {showFlatSKU && (
                  <FlatSkuScaleBarcodeCard form={sku} setField={setSkuField} currentSkuId={currentSku?.id ?? null} />
                )}

                {showFlatSKU && (
                  <FlatSkuWeightPoolCard form={sku} patch={patchSku} currentSkuId={currentSku?.id ?? null} />
                )}

                {showFlatSKU && (
                  <FlatSkuShippingCard form={sku} setField={setSkuField} patch={patchSku} />
                )}

              </div>

              {/* Sidebar column */}
              <ProductSidebar
                status={fields.status} onStatusChange={s => setField('status', s)}
                categoryId={fields.categoryId} onCategoryChange={id => setField('categoryId', id)}
                categories={categories} onAddCategory={handleAddCategory}
                onUpdateCategoryMargin={(id, margin) => onUpdateCategory(id, { name: categories.find(c => c.id === id)!.name, target_markup_basis_points: margin })}
              />
            </div>
          </form>
        </div>
        {activeTab === 'woocommerce' && savedProductId && connections.woo && (
          <div className="max-w-5xl mx-auto p-6">
            <ProductWooCommerceTab
              productId={savedProductId} connection={connections.woo} sku={currentSku}
              hasVariants={form.variantsMode} variants={detail.detail?.variants ?? []}
            />
          </div>
        )}
        {activeTab === 'bol' && savedProductId && connections.bol && (
          <div className="max-w-5xl mx-auto p-6">
            <ProductBolTab
              productId={savedProductId} connection={connections.bol} sku={currentSku} productName={form.existing?.name}
              onSaveBarcode={async (skuId, newBarcode) => {
                if (!currentSku) return
                await detail.updateSKU(skuId, {
                  ...skuPassThroughPayload(currentSku),
                  barcode: newBarcode || undefined,
                  tax_class_id: currentSku.tax_class_id,
                })
              }}
            />
          </div>
        )}
      </div>
    </div>
  )
}
