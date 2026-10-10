import { useState, useEffect, useCallback, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Plug } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import * as integrationsApi from '../integrations/api'
import type { Connection, ProductChannelListing, ProductChannelListingSyncStatus } from '../integrations/types'
import type { VariantWithSKUs, SKU } from './types'
import { useDateFormat } from '../format/DateFormatProvider'
import { apiErrorText } from '../api/apiErrorText'

// ── WooCommerce tab ───────────────────────────────────────────────────────────
//
// Per-channel sync state for this product on the connected WooCommerce
// store, fetched from GetProductChannelListingUseCase. A 404 (no listing
// row) is an expected, common state — most products aren't linked to every
// connected channel — rendered as "not linked yet", not an error.

// Brand name, not translated.
const BRAND = 'WooCommerce'

const WOO_SYNC_STATUS_STYLE: Record<ProductChannelListingSyncStatus, string> = {
  pending: 'text-warning',
  synced: 'text-success',
  error: 'text-danger',
}

export function ProductWooCommerceTab({
  productId, connection, sku, hasVariants, variants,
}: {
  productId: string; connection: Connection; sku: SKU | null
  hasVariants: boolean; variants: VariantWithSKUs[]
}) {
  const { formatDateTime } = useDateFormat()
  const { token } = useAuth()
  const { t } = useTranslation('pms')
  const syncLabel: Record<ProductChannelListingSyncStatus, string> = { pending: t('woo.pending'), synced: t('woo.synced'), error: t('woo.error') }
  const [listings, setListings] = useState<ProductChannelListing[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [linkSku, setLinkSku] = useState('')
  const [linking, setLinking] = useState(false)
  const [linkError, setLinkError] = useState<string | null>(null)

  const loadListings = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const result = await integrationsApi.listProductChannelListings(token, connection.id, productId)
      setListings(result)
    } catch (err) {
      setError(apiErrorText(err) ?? t('woo.loadFailed'))
    } finally {
      setLoading(false)
    }
  }, [token, connection.id, productId])

  useEffect(() => {
    loadListings()
  }, [loadListings])

  // Per-variant push (separate-products or variations mode) shows one row
  // per variant instead of collapsing everything to a single summary --
  // GetProductChannelListingUseCase's one-representative-row shape was
  // fine for a flat product but misleading otherwise. The "link existing"
  // form below only supports the flat (!HasVariants) case -- see
  // LinkExistingWooProductUseCase's own doc -- so it's only shown there.
  const isMultiVariant = hasVariants && variants.length > 1
  const listing = listings[0] ?? null
  const parentListing = listings.find(l => l.variant_id == null) ?? null

  const handleLinkExisting = async (e: FormEvent) => {
    e.preventDefault()
    if (!token || !linkSku.trim()) return
    setLinking(true)
    setLinkError(null)
    try {
      const result = await integrationsApi.linkExistingWooProduct(token, productId, connection.id, linkSku.trim())
      setListings([result])
      setLinkSku('')
    } catch (err) {
      setLinkError(apiErrorText(err) ?? t('woo.linkFailed'))
    } finally {
      setLinking(false)
    }
  }

  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-night-800/60 bg-gray-100 border border-line outline-none focus-visible:ring-2 focus-visible:ring-focus/60 cursor-not-allowed'
  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'

  return (
    <div data-testid="woocommerce-tab" className="rounded-xl border border-line p-4 space-y-3">
      <div className="flex items-center gap-2">
        <Plug size={13} className="text-ink-3" />
        <h3 className="text-xs font-semibold text-ink">{BRAND}</h3>
        <span className="text-2xs text-ink-4">· {connection.store_url}</span>
      </div>

      {loading && (
        <p className="text-2xs text-ink-4">{t('woo.loading')}</p>
      )}

      {!loading && error && (
        <p className="text-2xs text-danger">{error}</p>
      )}

      {!loading && !error && isMultiVariant && (
        <div className="space-y-2">
          {parentListing && (
            <WooListingRow
              label={t('woo.parent')} sku={null}
              listing={parentListing}
              syncLabel={syncLabel}
            />
          )}
          {variants.map(v => (
            <WooListingRow
              key={v.id}
              label={v.name}
              sku={v.skus?.[0]?.sku_code ?? null}
              listing={listings.find(l => l.variant_id === v.id) ?? null}
              syncLabel={syncLabel}
            />
          ))}
        </div>
      )}

      {!loading && !error && !isMultiVariant && !listing && (
        <div className="space-y-2">
          <p className="text-2xs text-ink-4">
            {t('woo.notLinkedHint')}
          </p>
          <form onSubmit={handleLinkExisting} className="flex items-center gap-2">
            <input
              type="text"
              value={linkSku}
              onChange={e => setLinkSku(e.target.value)}
              placeholder={t('woo.skuPlaceholder')}
              className="flex-1 px-3 py-2 text-xs rounded-lg bg-raised border border-line outline-none focus:ring-1 focus:ring-focus text-ink"
            />
            <button
              type="submit"
              disabled={linking || !linkSku.trim()}
              className="px-3 py-2 text-2xs font-medium rounded-lg bg-accent-solid hover:bg-accent-solid-hover disabled:opacity-50 text-white shrink-0"
            >
              {linking ? t('woo.linking') : t('woo.linkExisting')}
            </button>
          </form>
          {linkError && <p className="text-2xs text-danger">{linkError}</p>}
        </div>
      )}

      {!loading && !error && !isMultiVariant && listing && (
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className={labelClass}>{t('woo.productId')}</label>
            <input disabled readOnly value={listing.external_id} className={clsx(inputClass, 'text-ink-2')} />
          </div>
          <div>
            <label className={labelClass}>{t('woo.syncStatus')}</label>
            <input
              disabled
              readOnly
              value={syncLabel[listing.sync_status]}
              className={clsx(inputClass, WOO_SYNC_STATUS_STYLE[listing.sync_status])}
            />
          </div>
          <div>
            <label className={labelClass}>{t('woo.lastSynced')}</label>
            <input
              disabled
              readOnly
              value={formatDateTime(listing.last_synced_at)}
              className={clsx(inputClass, 'text-ink-2')}
            />
          </div>
        </div>
      )}

      {/* woo_tax_class_slug is set once by import and never cleared, so it
          persists independently of listing (e.g. after "Delete imported
          products" removes the listing but not this SKU) -- shown whenever
          this SKU actually came from Woo, not gated on listing being present. */}
      {sku?.woo_tax_class_slug != null && (
        <div className="grid grid-cols-2 gap-3 pt-3 border-t border-line">
          <div>
            <label htmlFor="product-woo-tax-class" className={labelClass}>{t('woo.taxClass')}</label>
            <input
              id="product-woo-tax-class"
              disabled
              readOnly
              value={sku.woo_tax_class_slug || t('woo.standard')}
              className={clsx(inputClass, 'text-ink-2')}
            />
          </div>
          <div>
            <label htmlFor="product-woo-tax-status" className={labelClass}>{t('woo.taxStatus')}</label>
            <input
              id="product-woo-tax-status"
              disabled
              readOnly
              value={sku.woo_tax_status || '—'}
              className={clsx(inputClass, 'text-ink-2')}
            />
          </div>
        </div>
      )}
    </div>
  )
}

// One row of the multi-variant WooCommerce sync summary -- a variant (or
// the variable parent) alongside whatever listing matches it, if any.
function WooListingRow({ label, sku, listing, syncLabel }: {
  label: string; sku: string | null; listing: ProductChannelListing | null; syncLabel: Record<ProductChannelListingSyncStatus, string>
}) {
  const { t } = useTranslation('pms')
  return (
    <div className="flex items-center gap-3 rounded-lg px-3 py-2 dark:bg-night-800/40 bg-gray-50 text-2xs">
      <div className="flex-1 min-w-0">
        <div className="font-medium truncate text-ink">{label}</div>
        {sku && <div className="text-ink-4 truncate">{sku}</div>}
      </div>
      {listing ? (
        <>
          <span className="text-ink-3 shrink-0">{t('woo.wooId', { id: listing.external_id })}</span>
          <span className={clsx('font-medium shrink-0', WOO_SYNC_STATUS_STYLE[listing.sync_status])}>
            {syncLabel[listing.sync_status]}
          </span>
        </>
      ) : (
        <span className="text-ink-4 shrink-0">{t('woo.notLinked')}</span>
      )}
    </div>
  )
}
