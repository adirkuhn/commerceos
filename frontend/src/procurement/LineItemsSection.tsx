import { useState, useEffect, useRef, Fragment } from 'react'
import { Plus, Trash2, Edit3, Search, Upload, Sparkles, Loader2 } from 'lucide-react'
import clsx from 'clsx'
import { Trans, useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { pollParsingJob } from './pollParsingJob'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import * as procurementApi from './api'
import * as integrationsApi from '../integrations/api'
import type { PurchaseOrderLineItem, AddLineItemInput, UpdateLineItemInput } from './types'
import type { SKU } from '../pms/types'
import { ProductSkuPicker } from '../components/common/ProductSkuPicker'
import { useConfirm } from '../components/common/useConfirm'
import { friendlyParsingError, lineTypeLabel, isChargeLineType, signedLineItemAmount, unitsPerPurchasedItemOnLink } from './procurementHelpers'
import { LineItemFormRow } from './LineItemFormRow'
import { formatMoney } from '../format/money'

// ── Line Items ───────────────────────────────────────────────────────────────

interface LineItemsSectionProps {
  purchaseOrderId: string
  lineItems: PurchaseOrderLineItem[]
  currencyCode: string
  // Set only right after a document-upload create -- picked up once (on
  // mount) to poll that already-in-flight parse instead of the create flow
  // blocking on it.
  initialParsingJobId?: string | null
  onAdd: (input: AddLineItemInput) => Promise<void>
  onUpdate: (id: string, input: UpdateLineItemInput) => Promise<void>
  onRemove: (id: string) => Promise<void>
  onRefresh: () => void
}

export function LineItemsSection({ purchaseOrderId, lineItems, currencyCode, initialParsingJobId, onAdd, onUpdate, onRemove, onRefresh }: LineItemsSectionProps) {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { t } = useTranslation(['procurement', 'common'])
  const [adding, setAdding] = useState(false)
  const [editingLineId, setEditingLineId] = useState<string | null>(null)
  const [reviewingLineId, setReviewingLineId] = useState<string | null>(null)
  const [uploadStatus, setUploadStatus] = useState<'idle' | 'uploading' | 'parsing'>(initialParsingJobId ? 'parsing' : 'idle')
  const [parsingSlow, setParsingSlow] = useState(false)
  const [uploadError, setUploadError] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Picks up a parse that was already kicked off by the create-from-document
  // flow -- runs once (initialParsingJobId only has a value on the very
  // first mount right after that create, and this component isn't reused
  // across different POs without unmounting first).
  useEffect(() => {
    if (!initialParsingJobId) return
    let cancelled = false
    pollParsingJob(
      () => procurementApi.getDocumentParsingJob(token ?? '', initialParsingJobId),
      { onSlow: () => { if (!cancelled) setParsingSlow(true) } },
    )
      .then(finalJob => {
        if (cancelled) return
        if (finalJob.status === 'failed') {
          setUploadError(friendlyParsingError(finalJob.error_message))
        }
        onRefresh()
      })
      .catch(err => {
        if (!cancelled) setUploadError(err instanceof Error ? err.message : t('parsing.failed'))
      })
      .finally(() => {
        if (!cancelled) { setUploadStatus('idle'); setParsingSlow(false) }
      })
    return () => { cancelled = true }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialParsingJobId])

  async function handleRemove(id: string) {
    if (!(await confirm({ title: t('lines.removeConfirm'), confirmLabel: t('lines.removeConfirmLabel'), destructive: true }))) return
    await onRemove(id)
  }

  async function handleFileSelected(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return

    // Only PDFs need AI to parse (CSV/XLSX go through the deterministic
    // path) -- warn up front rather than let the upload silently fail to
    // extract anything, so the user knows manual entry is the fallback.
    const isPDF = file.type === 'application/pdf' || file.name.toLowerCase().endsWith('.pdf')
    if (isPDF) {
      const aiConnection = await integrationsApi.getAIConnection(token ?? '').catch(() => null)
      if (!aiConnection && !(await confirm({
        title: t('lines.noAiTitle'),
        message: t('lines.noAiMessage'),
        confirmLabel: t('lines.uploadAnyway'),
      }))) {
        return
      }
    }

    setUploadError('')
    setUploadStatus('uploading')
    try {
      const job = await procurementApi.attachDocumentToPurchaseOrder(token ?? '', purchaseOrderId, file)
      setUploadStatus('parsing')
      const finalJob = await pollParsingJob(
        () => procurementApi.getDocumentParsingJob(token ?? '', job.id),
        { onSlow: () => setParsingSlow(true) },
      )
      if (finalJob.status === 'failed') {
        setUploadError(friendlyParsingError(finalJob.error_message))
      }
      // Always refresh, even on a failed parse -- the document itself was
      // still saved and should show up in the Documents section so it can
      // be opened for manual entry.
      onRefresh()
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : t('lines.uploadFailed'))
    } finally {
      setUploadStatus('idle')
      setParsingSlow(false)
    }
  }

  return (
    <div className="rounded-xl border border-line overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 dark:bg-night-900/50 bg-gray-50 border-b border-line">
        <h3 className="text-xs font-semibold text-ink">{t('lines.title')}</h3>
        <div className="flex items-center gap-2">
          {uploadStatus === 'uploading' && (
            <span className="flex items-center gap-1 text-2xs text-ink-4">
              <Loader2 size={11} className="animate-spin" /> {t('lines.uploading')}
            </span>
          )}
          {uploadStatus === 'parsing' && (
            <span className="flex items-center gap-1 text-2xs text-warning">
              <Loader2 size={11} className="animate-spin" />
              {parsingSlow ? t('lines.stillParsing') : t('lines.parsing')}
            </span>
          )}
          <input ref={fileInputRef} type="file" aria-label={t('lines.uploadDocument')} accept=".pdf,.csv,.xlsx" className="hidden" onChange={handleFileSelected} />
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploadStatus !== 'idle'}
            className="flex items-center gap-1 px-2.5 py-1 rounded-lg border border-line bg-raised text-2xs text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
          >
            <Upload size={11} /> {t('lines.uploadDocument')}
          </button>
          {!adding && (
            <button
              type="button"
              onClick={() => setAdding(true)}
              className="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover"
            >
              <Plus size={11} /> {t('lines.addLine')}
            </button>
          )}
        </div>
      </div>
      {uploadError && (
        <p className="text-2xs text-danger bg-red-50 dark:bg-red-500/10 px-4 py-1.5">{uploadError}</p>
      )}

      <table className="w-full text-xs">
        <thead>
          <tr className="dark:bg-night-900/30 bg-gray-50">
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">{t('lines.colDescription')}</th>
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">{t('lines.colType')}</th>
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">{t('lines.colLinkedSku')}</th>
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">{t('lines.colBatch')}</th>
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">{t('lines.colQty')}</th>
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">{t('lines.colUnitCost')}</th>
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">{t('lines.colTotal')}</th>
            <th className="px-4 py-2 w-20" />
          </tr>
        </thead>
        <tbody className="divide-y divide-line/50">
          {lineItems.length === 0 && !adding && (
            <tr><td colSpan={8} className="px-4 py-6 text-center text-ink-4">{t('lines.empty')}</td></tr>
          )}
          {lineItems.map(li => {
            const isCharge = isChargeLineType(li.line_type)
            return editingLineId === li.id ? (
              <LineItemFormRow
                key={li.id}
                initial={li}
                currencyCode={currencyCode}
                onSubmit={async input => { await onUpdate(li.id, input); setEditingLineId(null) }}
                onCancel={() => setEditingLineId(null)}
              />
            ) : (
              <Fragment key={li.id}>
                <tr className={clsx('transition-colors', li.sku_id || isCharge ? 'hover:bg-hover/30' : 'dark:bg-amber-500/5 bg-amber-50/40')}>
                  <td className="px-4 py-2.5 text-ink">{li.raw_description}</td>
                  <td className="px-4 py-2.5 text-ink-3">
                    {isCharge && (
                      <div className="flex items-center gap-1.5">
                        <span className="text-2xs">{lineTypeLabel(li.line_type)}</span>
                        {!li.line_type_confirmed && (
                          <button
                            type="button"
                            onClick={() => onUpdate(li.id, {
                              sku_id: li.sku_id ?? null, raw_sku: li.raw_sku, raw_description: li.raw_description,
                              quantity: li.quantity, unit_cost_minor_units: li.unit_cost_minor_units, line_type: li.line_type,
                            })}
                            title={t('lines.confirmTypeTitle')}
                            className="text-2xs font-medium text-accent hover:underline"
                          >
                            {t('lines.confirm')}
                          </button>
                        )}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-2.5 text-ink-3">
                    {isCharge ? (
                      <span className="text-2xs">—</span>
                    ) : li.sku_id ? (
                      li.product_name ? (
                        <span className="text-2xs">
                          {li.product_name} <span className="font-mono text-ink-4">— {li.sku_code}</span>
                          {li.units_per_purchased_item > 1 && (
                            <span className="ml-1 text-warning" title={t('lines.unitsPerItemTitle')}>
                              × {li.units_per_purchased_item}
                            </span>
                          )}
                        </span>
                      ) : (
                        <span className="font-mono text-2xs">{li.sku_id}</span>
                      )
                    ) : (
                      <button
                        type="button"
                        onClick={() => setReviewingLineId(reviewingLineId === li.id ? null : li.id)}
                        className="flex items-center gap-1 text-2xs font-medium text-warning hover:underline"
                      >
                        <Sparkles size={10} /> {t('lines.needsReview')}
                      </button>
                    )}
                  </td>
                  <td className="px-4 py-2.5 text-ink-3">
                    {li.suggested_lot_number ? (
                      <span className="text-2xs" title={t('lines.lotTitle')}>
                        {li.suggested_lot_number}
                        {li.suggested_expiry_date && (
                          <span className="text-ink-4">{t('lines.exp', { date: li.suggested_expiry_date })}</span>
                        )}
                      </span>
                    ) : (
                      <span className="text-2xs">—</span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 text-ink">{li.quantity}</td>
                  <td className="px-4 py-2.5 text-ink">{formatMoney(li.unit_cost_minor_units, currencyCode)}</td>
                  <td className="px-4 py-2.5 text-ink font-medium">
                    {formatMoney(signedLineItemAmount(li), currencyCode)}
                  </td>
                  <td className="px-4 py-2.5">
                    <div className="flex items-center gap-1">
                      <button aria-label={t('common:actions.edit')}
                        type="button"
                        onClick={() => setEditingLineId(li.id)}
                        className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                        title={t('common:actions.edit')}
                      >
                        <Edit3 size={12} className="text-ink-3" />
                      </button>
                      <button aria-label={t('common:actions.remove')}
                        type="button"
                        onClick={() => handleRemove(li.id)}
                        className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                        title={t('common:actions.remove')}
                      >
                        <Trash2 size={12} className="text-ink-3 hover:text-danger" />
                      </button>
                    </div>
                  </td>
                </tr>
                {reviewingLineId === li.id && (
                  <tr>
                    <td colSpan={8} className="px-4 pb-3 dark:bg-amber-500/5 bg-amber-50/40">
                      <ReviewUnmatchedLineItem
                        lineItem={li}
                        onResolved={() => { setReviewingLineId(null); onRefresh() }}
                        onCancel={() => setReviewingLineId(null)}
                      />
                    </td>
                  </tr>
                )}
              </Fragment>
            )
          })}
          {adding && (
            <LineItemFormRow
              currencyCode={currencyCode}
              onSubmit={async input => { await onAdd(input); setAdding(false) }}
              onCancel={() => setAdding(false)}
            />
          )}
          {!adding && (
            <tr>
              <td colSpan={8} className="px-4 py-2">
                <button
                  type="button"
                  onClick={() => setAdding(true)}
                  className="flex items-center gap-1 text-2xs font-medium text-accent hover:underline"
                >
                  <Plus size={11} /> {t('lines.addLine')}
                </button>
              </td>
            </tr>
          )}
        </tbody>
        {lineItems.length > 0 && (
          <tfoot>
            <tr className="border-t border-line font-medium">
              <td colSpan={6} className="px-4 py-2.5 text-right text-ink-3 uppercase tracking-wider text-2xs">
                {t('lines.total')}
              </td>
              <td className="px-4 py-2.5 text-ink">
                {formatMoney(lineItems.reduce((sum, li) => sum + signedLineItemAmount(li), 0), currencyCode)}
              </td>
              <td className="px-4 py-2.5" />
            </tr>
          </tfoot>
        )}
      </table>
      {confirmDialog}
    </div>
  )
}

// ── Review unmatched line item ──────────────────────────────────────────────

interface ReviewUnmatchedLineItemProps {
  lineItem: PurchaseOrderLineItem
  onResolved: () => void
  onCancel: () => void
}

function ReviewUnmatchedLineItem({ lineItem, onResolved, onCancel }: ReviewUnmatchedLineItemProps) {
  const { token } = useAuth()
  const { settings: tenantSettings } = useTenantSettings(token ?? '')
  const [mode, setMode] = useState<'choose' | 'link' | 'create'>('choose')
  const [productName, setProductName] = useState(lineItem.raw_description)
  const [skuCode, setSkuCode] = useState(lineItem.raw_sku)
  const [price, setPrice] = useState('0.00')
  const [unitsPerPurchasedItem, setUnitsPerPurchasedItem] = useState('1')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const { t } = useTranslation(['procurement', 'common'])

  async function handleLink(sku: SKU) {
    setError('')
    setSaving(true)
    try {
      await procurementApi.linkLineItemToSku(token ?? '', lineItem.id, {
        sku_id: sku.id, units_per_purchased_item: parseInt(unitsPerPurchasedItemOnLink(sku, unitsPerPurchasedItem) || '1', 10),
      })
      onResolved()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('review.linkFailed'))
      setSaving(false)
    }
  }

  async function handleCreateProduct(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setSaving(true)
    try {
      await procurementApi.createProductAndLinkLineItem(token ?? '', lineItem.id, {
        product_name: productName, sku_code: skuCode,
        currency_code: tenantSettings?.default_currency || 'USD',
        price_minor_units: Math.round(parseFloat(price || '0') * 100),
      })
      onResolved()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('review.createFailed'))
      setSaving(false)
    }
  }

  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-night-800 bg-white border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'
  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1'

  return (
    <div className="rounded-lg border dark:border-amber-500/30 border-amber-300 dark:bg-night-950 bg-white p-3 space-y-2">
      <div className="text-2xs text-ink-3">
        {lineItem.raw_sku
          ? <Trans t={t} i18nKey="review.extractedWithSku" values={{ sku: lineItem.raw_sku }} components={{ code: <span className="font-mono" /> }} />
          : t('review.extracted')}
      </div>

      {mode === 'choose' && (
        <div className="flex items-center gap-2">
          <button type="button" onClick={() => setMode('link')}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover">
            <Search size={10} /> {t('review.linkExisting')}
          </button>
          <button type="button" onClick={() => setMode('create')}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-fill text-2xs text-ink-2 font-medium hover:bg-hover">
            <Plus size={10} /> {t('review.createNew')}
          </button>
          <button type="button" onClick={onCancel} className="text-2xs text-ink-4 hover:underline ml-auto">
            {t('review.dismiss')}
          </button>
        </div>
      )}

      {mode === 'link' && (
        <div className="space-y-2">
          <div>
            <label htmlFor={`review-units-per-item-${lineItem.id}`} className={labelClass}>
              {t('review.unitsPerItem')}
            </label>
            <input
              id={`review-units-per-item-${lineItem.id}`} type="number" min="1" step="1"
              value={unitsPerPurchasedItem} onChange={e => setUnitsPerPurchasedItem(e.target.value)}
              title={t('review.unitsPerItemTitle')}
              className={clsx(inputClass, 'w-24')}
            />
          </div>
          <ProductSkuPicker onSelect={sku => handleLink(sku)} onClose={() => setMode('choose')} />
        </div>
      )}

      {mode === 'create' && (
        <form onSubmit={handleCreateProduct} className="space-y-2">
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label htmlFor={`review-name-${lineItem.id}`} className={labelClass}>{t('review.productName')}</label>
              <input id={`review-name-${lineItem.id}`} required value={productName} onChange={e => setProductName(e.target.value)} className={inputClass} />
            </div>
            <div>
              <label htmlFor={`review-sku-${lineItem.id}`} className={labelClass}>{t('review.skuCode')}</label>
              <input id={`review-sku-${lineItem.id}`} required value={skuCode} onChange={e => setSkuCode(e.target.value)} className={inputClass} />
            </div>
            <div>
              <label htmlFor={`review-price-${lineItem.id}`} className={labelClass}>{t('review.price')}</label>
              <input id={`review-price-${lineItem.id}`} type="number" min="0" step="0.01" value={price} onChange={e => setPrice(e.target.value)} className={inputClass} />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button type="submit" disabled={saving}
              className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-2xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50">
              {saving ? t('review.creating') : t('review.createLink')}
            </button>
            <button type="button" onClick={() => setMode('choose')} className="text-2xs text-ink-4 hover:underline">
              {t('review.back')}
            </button>
          </div>
        </form>
      )}

      {error && <p className="text-2xs text-red-400">{error}</p>}
    </div>
  )
}
