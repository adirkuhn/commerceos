import i18next from 'i18next'
import { useState } from 'react'
import * as pmsApi from '../api'
import type { Product, ProductStatus, VariantWithSKUs } from '../types'
import type { TenantSettings } from '../../settings/types'
import { useProductDetail } from './useProductDetail'
import { productFieldsFromProduct, type ProductFields, type SetProductField } from '../productFields'
import { EMPTY_FLAT_SKU_FORM, flatSkuBasePayload, flatSkuCreatePayload, flatSkuFormFromSKU, flatSkuUpdatePayload, type FlatSkuForm, type PatchFlatSku, type SetFlatSkuField } from '../flatSkuForm'

export interface ProductWritePayload {
  name: string
  slug?: string
  description?: string
  category_id?: string | null
  status: ProductStatus
  has_variants?: boolean
  sync_variants_as_woo_variations?: boolean
  tags?: string[]
}

interface UseProductEditFormArgs {
  token: string | null
  productId: string | null
  tenantSettings: TenantSettings | null
  onCreate: (payload: ProductWritePayload) => Promise<Product>
  onUpdate: (id: string, payload: ProductWritePayload) => Promise<Product>
}

// All of the product editor's form state -- product fields, the flat
// single-SKU form, the variants-mode switch -- plus hydrating it from the
// loaded product and saving it back.
export function useProductEditForm({ token, productId, tenantSettings, onCreate, onUpdate }: UseProductEditFormArgs) {
  const [savedProductId, setSavedProductId] = useState<string | null>(productId)
  const detail = useProductDetail(token ?? '', savedProductId)
  const existing = detail.detail?.product

  const [fields, setFields] = useState<ProductFields>(() => productFieldsFromProduct(existing))
  const setField: SetProductField = (key, value) => setFields(prev => ({ ...prev, [key]: value }))
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

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
  const [sku, setSku] = useState<FlatSkuForm>(EMPTY_FLAT_SKU_FORM)
  const setSkuField: SetFlatSkuField = (key, value) => setSku(prev => ({ ...prev, [key]: value }))
  const patchSku: PatchFlatSku = patch => setSku(prev => ({ ...prev, ...(typeof patch === 'function' ? patch(prev) : patch) }))

  // Sync local form state once the product detail loads (edit mode) — runs once.
  const [hydrated, setHydrated] = useState(false)
  if (!hydrated && existing) {
    setFields(productFieldsFromProduct(existing))
    setVariantsMode(existing.has_variants)
    setSyncVariantsAsWooVariations(existing.sync_variants_as_woo_variations ?? false)
    const variants = detail.detail?.variants ?? []
    const loadedSku = variants.length === 1 ? variants[0].skus?.[0] : undefined
    if (loadedSku) patchSku(flatSkuFormFromSKU(loadedSku))
    setHydrated(true)
  }

  // A brand-new product (productId===null, so nothing to hydrate from
  // above) still needs its currency seeded from the tenant's own default
  // rather than sitting on the initial 'USD' forever -- gated on
  // tenantSettings actually having loaded (not just !loading, which is
  // still false for an instant before the fetch effect kicks in) so this
  // doesn't fire-and-mark-done before the real value arrives.
  const [currencyDefaulted, setCurrencyDefaulted] = useState(false)
  if (!currencyDefaulted && productId === null && tenantSettings) {
    if (tenantSettings.default_currency) setSkuField('currency', tenantSettings.default_currency)
    setCurrencyDefaulted(true)
  }

  const singleVariant: VariantWithSKUs | null =
    !variantsMode && detail.detail?.variants.length === 1 ? detail.detail.variants[0] : null
  const currentSku = singleVariant?.skus?.[0] ?? null
  const showFlatSKU = !variantsMode && (savedProductId ? singleVariant !== null || detail.detail?.variants.length === 0 : true)

  const productPayload = (): ProductWritePayload => ({
    name: fields.name, description: fields.description, category_id: fields.categoryId || null, status: fields.status,
    has_variants: variantsMode, sync_variants_as_woo_variations: syncVariantsAsWooVariations, tags: fields.tags,
    // Only meaningful on update -- create always derives slug from
    // name server-side, and only sending it when it actually changed
    // from what loaded avoids a spurious uniqueness check against the
    // product's own current slug.
    ...(savedProductId && fields.slug !== existing?.slug ? { slug: fields.slug } : {}),
  })

  const createProduct = async () => {
    const product = await onCreate(productPayload())
    if (!variantsMode) {
      const variant = await pmsApi.createVariant(token ?? '', product.id, { name: product.name })
      if (sku.skuCode) await pmsApi.createSKU(token ?? '', variant.id, flatSkuCreatePayload(sku))
    }
    // Set this last, after the variant/SKU writes above complete --
    // useProductDetail's auto-fetch fires the instant `savedProductId`
    // changes (its `productId` prop), so setting it any earlier races
    // that fetch against the still-in-flight creates above and can
    // land on an empty `variants: []` permanently (no later refetch
    // ever corrects it, since nothing besides this transition
    // triggers one).
    setSavedProductId(product.id)
  }

  const updateProduct = async (id: string) => {
    const updatedProduct = await onUpdate(id, productPayload())
    // The `hydrated` sync-from-`existing` block above only ever runs
    // once (by design, so it doesn't clobber in-progress edits every
    // time `existing` changes) -- so a server-side transform like slug
    // regeneration would otherwise sit in `detail` unseen until a full
    // page reload. Patch every field the save just touched directly
    // from the response, which -- unlike a subsequent GET -- already
    // reflects transforms UpdateProductUseCase applied in memory before
    // persisting (e.g. slug regeneration/disambiguation).
    setFields(productFieldsFromProduct(updatedProduct))
    // addSKU/updateSKU below already refresh `detail` themselves, but a
    // save with no SKU changes wouldn't otherwise touch it -- without
    // this, `existing` (used for the slug-changed diff above) stays
    // stale for a second save within the same edit session.
    detail.refresh()
    if (variantsMode) return
    if (singleVariant) {
      const loadedSku = singleVariant.skus?.[0]
      if (loadedSku) {
        await detail.updateSKU(loadedSku.id, flatSkuUpdatePayload(sku, loadedSku))
      } else if (sku.skuCode) {
        await detail.addSKU(singleVariant.id, flatSkuBasePayload(sku))
      }
    } else if (sku.skuCode) {
      const variant = await pmsApi.createVariant(token ?? '', id, { name: fields.name })
      await pmsApi.createSKU(token ?? '', variant.id, flatSkuBasePayload(sku))
      detail.refresh()
    }
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      if (!savedProductId) await createProduct()
      else await updateProduct(savedProductId)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : i18next.t('pms:errors.saveProduct'))
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
      await onUpdate(savedProductId, { name: fields.name, description: fields.description, category_id: fields.categoryId || null, status: fields.status, has_variants: true, tags: fields.tags })
      setVariantsMode(true)
    } catch (err) {
      setEnableVariantsError(err instanceof Error ? err.message : i18next.t('pms:errors.enableVariants'))
    } finally {
      setEnablingVariants(false)
    }
  }

  return {
    savedProductId, detail, existing,
    fields, setField,
    sku, setSkuField, patchSku,
    variantsMode, setVariantsMode, syncVariantsAsWooVariations, setSyncVariantsAsWooVariations,
    singleVariant, currentSku, showFlatSKU,
    saving, formError, setFormError, handleSave,
    enablingVariants, enableVariantsError, handleEnableVariants,
  }
}
