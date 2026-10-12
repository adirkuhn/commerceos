import { useState, useEffect, useRef } from 'react'
import { ArrowLeft, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { usePurchaseOrderDetail } from './hooks/usePurchaseOrderDetail'
import { useSuppliers } from '../suppliers/hooks/useSuppliers'
import type { Supplier } from '../suppliers/types'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import * as procurementApi from './api'
import type { PurchaseOrder, PurchaseOrderStatus, CreatePurchaseOrderInput, UpdatePurchaseOrderInput } from './types'
import { CURRENCIES, poStatusLabel, todayDateString } from './procurementHelpers'
import { DocumentsSection } from './DocumentsSection'
import { LineItemsSection } from './LineItemsSection'
import { ReceivingSection } from './ReceivingSection'
import { apiErrorText } from '../api/apiErrorText'

// ── Purchase Order Edit Page ────────────────────────────────────────────────

interface PurchaseOrderEditPageProps {
  purchaseOrderId: string | null
  // Set only right after a document-upload create -- lets LineItemsSection
  // pick up the still-in-flight parse and poll it instead of showing an
  // empty list until the user manually refreshes.
  initialParsingJobId?: string | null
  onCreate: (input: CreatePurchaseOrderInput) => Promise<PurchaseOrder>
  // Called with the new id right after a create, so the URL can switch from
  // /procurement/new to the PO's own URL.
  onCreated: (id: string) => void
  onClose: () => void
}

export function PurchaseOrderEditPage({ purchaseOrderId, initialParsingJobId, onCreate, onCreated, onClose }: PurchaseOrderEditPageProps) {
  const { token } = useAuth()
  const { suppliers } = useSuppliers(token ?? '')
  const { t } = useTranslation(['procurement', 'common'])
  const { warehouses } = useWarehouses(token ?? '')
  const { settings: tenantSettings } = useTenantSettings(token ?? '')
  const [createdId, setCreatedId] = useState<string | null>(null)
  const effectiveId = purchaseOrderId ?? createdId
  const { detail, error: detailError, refresh: refreshDetail, updateHeader, addLineItem, updateLineItem, removeLineItem, deleteDocument } = usePurchaseOrderDetail(token ?? '', effectiveId)

  const po = detail?.purchase_order ?? null

  // AI-suggested supplier name matched against the existing supplier list --
  // when it matches, the dropdown pre-selects it and a Confirm control takes
  // over instead of the "create a new supplier" box (see MissingSupplierReview).
  const suggestedSupplierMatch = po?.suggested_supplier_name
    ? suppliers.find(s => normalizeSupplierName(s.name) === normalizeSupplierName(po.suggested_supplier_name!))
    : undefined

  const [supplierId, setSupplierId] = useState(po?.supplier_id ?? '')
  const [warehouseId, setWarehouseId] = useState(po?.warehouse_id ?? '')
  const [status, setStatus] = useState<PurchaseOrderStatus>(po?.status ?? 'draft')
  const [currencyCode, setCurrencyCode] = useState(po?.currency_code ?? 'USD')
  const [expectedDate, setExpectedDate] = useState(purchaseOrderId === null ? todayDateString() : (po?.expected_date ?? ''))
  const [notes, setNotes] = useState(po?.notes ?? '')
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  // Seeds the currency from the tenant's system-settings default for a
  // brand-new PO only -- an existing PO's own currency_code (set below)
  // always wins once `detail` loads.
  const currencyTouched = useRef(false)
  useEffect(() => {
    if (!po && !currencyTouched.current && tenantSettings?.default_currency) {
      setCurrencyCode(tenantSettings.default_currency)
    }
  }, [po, tenantSettings])

  // Header fields start empty for an existing PO because `detail` loads
  // asynchronously after mount. Seed the form once per PO id — a ref (not
  // state) so it doesn't retrigger this effect — so later refreshes (e.g.
  // after adding a line item) don't clobber in-progress header edits.
  const prefilledForId = useRef<string | null>(null)
  useEffect(() => {
    if (po && prefilledForId.current !== po.id) {
      prefilledForId.current = po.id
      setSupplierId(po.supplier_id ?? '')
      setWarehouseId(po.warehouse_id ?? '')
      setStatus(po.status)
      setCurrencyCode(po.currency_code)
      setExpectedDate(po.expected_date ?? '')
      setNotes(po.notes)
    }
  }, [po])

  const isNew = purchaseOrderId === null && createdId === null

  // Pre-selects the dropdown once the AI suggestion resolves to an existing
  // supplier -- separate from the field-seeding effect above because
  // `suppliers` loads on its own async timer, often after that effect's
  // one-shot-per-PO guard has already fired. Guarded the same way so it
  // doesn't fight a choice the user already made.
  const suggestedSupplierPreselectedFor = useRef<string | null>(null)
  useEffect(() => {
    if (!po || po.supplier_id || !suggestedSupplierMatch) return
    if (suggestedSupplierPreselectedFor.current === po.id) return
    suggestedSupplierPreselectedFor.current = po.id
    setSupplierId(suggestedSupplierMatch.id)
  }, [po, suggestedSupplierMatch])

  const [confirmingSuggestedSupplier, setConfirmingSuggestedSupplier] = useState(false)
  async function handleConfirmSuggestedSupplier() {
    if (!effectiveId || !suggestedSupplierMatch) return
    setFormError('')
    setConfirmingSuggestedSupplier(true)
    try {
      const updated = await procurementApi.createSupplierAndAssignToPurchaseOrder(token ?? '', effectiveId, { name: suggestedSupplierMatch.name })
      if (updated.supplier_id) setSupplierId(updated.supplier_id)
      await refreshDetail()
    } catch (err) {
      setFormError(apiErrorText(err) ?? t('po.confirmSupplierFailed'))
    } finally {
      setConfirmingSuggestedSupplier(false)
    }
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      if (isNew) {
        const input: CreatePurchaseOrderInput = {
          supplier_id: supplierId, warehouse_id: warehouseId.trim() === '' ? null : warehouseId, currency_code: currencyCode,
          expected_date: expectedDate.trim() === '' ? null : expectedDate,
          notes,
        }
        const created = await onCreate(input)
        setCreatedId(created.id)
        onCreated(created.id)
      } else if (effectiveId) {
        const input: UpdatePurchaseOrderInput = {
          supplier_id: supplierId, warehouse_id: warehouseId.trim() === '' ? null : warehouseId, status, currency_code: currencyCode,
          expected_date: expectedDate.trim() === '' ? null : expectedDate,
          notes,
        }
        await updateHeader(input)
      }
    } catch (err) {
      setFormError(apiErrorText(err) ?? t('po.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  // Called from the Receiving section's "Receive stock" form the first
  // time it needs a warehouse and the PO doesn't have one yet -- persists
  // it via the same updateHeader path the header Save button uses, and
  // mirrors it into the header select immediately so it's already correct
  // if the header form gets saved later in the same session.
  async function handleAssignWarehouse(newWarehouseId: string) {
    if (!effectiveId) return
    await updateHeader({
      supplier_id: supplierId, warehouse_id: newWarehouseId, status, currency_code: currencyCode,
      expected_date: expectedDate.trim() === '' ? null : expectedDate, notes,
    })
    setWarehouseId(newWarehouseId)
  }

  const labelClass = 'block text-2xs font-medium text-ink-4 caps-label mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> {t('po.back')}
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {isNew ? t('po.new') : t('po.edit')}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
          >
            {t('po.discard')}
          </button>
          <button
            type="submit" form="po-form" disabled={saving || !supplierId}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? t('common:actions.saving') : isNew ? t('po.create') : t('common:actions.save')}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto p-6 space-y-5">
          <form id="po-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border border-line p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-ink">{t('po.general')}</h3>
                {!isNew && (
                  <select
                    aria-label={t('po.status')}
                    value={status}
                    onChange={e => setStatus(e.target.value as PurchaseOrderStatus)}
                    className="px-2 py-1 text-2xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60 capitalize"
                  >
                    {(['draft', 'sent', 'partially_received', 'received', 'cancelled', 'needs_review'] as const).map(s => (
                      <option key={s} value={s} className="capitalize">{poStatusLabel(s)}</option>
                    ))}
                  </select>
                )}
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <div className="flex items-center justify-between mb-1.5">
                    <label htmlFor="po-supplier" className={labelClass + ' mb-0'}>{t('po.supplier')}</label>
                    {suggestedSupplierMatch && !po?.supplier_id && (
                      <button
                        type="button" onClick={handleConfirmSuggestedSupplier} disabled={confirmingSuggestedSupplier}
                        className="text-2xs font-semibold text-accent hover:underline disabled:opacity-50"
                      >
                        {confirmingSuggestedSupplier ? t('po.confirming') : t('po.confirm')}
                      </button>
                    )}
                  </div>
                  <select
                    id="po-supplier" required value={supplierId} onChange={e => setSupplierId(e.target.value)}
                    className={inputClass}
                  >
                    <option value="">{t('po.selectSupplier')}</option>
                    {suppliers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                  </select>
                  {suggestedSupplierMatch && !po?.supplier_id && (
                    <p className="text-2xs text-accent mt-1">{t('po.suggestedConfirm')}</p>
                  )}
                </div>
                <div>
                  <label htmlFor="po-warehouse" className={labelClass}>{t('po.warehouseOptional')}</label>
                  <select
                    id="po-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)}
                    className={inputClass}
                  >
                    <option value="">{t('po.warehouseNotSet')}</option>
                    {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="po-currency" className={labelClass}>{t('po.currency')}</label>
                  <select
                    id="po-currency" required value={currencyCode} onChange={e => { currencyTouched.current = true; setCurrencyCode(e.target.value) }}
                    className={inputClass}
                  >
                    {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="po-expected-date" className={labelClass}>{t('po.expectedDate')}</label>
                  <input
                    id="po-expected-date" type="date" value={expectedDate ?? ''} onChange={e => setExpectedDate(e.target.value)}
                    className={inputClass}
                  />
                </div>
              </div>
              <div>
                <label htmlFor="po-notes" className={labelClass}>{t('po.notes')}</label>
                <textarea
                  id="po-notes" rows={3} value={notes} onChange={e => setNotes(e.target.value)}
                  className={inputClass}
                />
              </div>
              {(formError || detailError) && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError || detailError}</p>
              )}
            </div>
          </form>

          {!isNew && effectiveId && po && !po.supplier_id && !suggestedSupplierMatch && (
            <MissingSupplierReview
              purchaseOrderId={effectiveId}
              suggestedName={po.suggested_supplier_name ?? ''}
              suppliers={suppliers}
              onAssigned={supplierId => { setSupplierId(supplierId); refreshDetail() }}
            />
          )}

          {!isNew && effectiveId && (detail?.documents.length ?? 0) > 0 && (
            <DocumentsSection documents={detail!.documents} onDelete={deleteDocument} onRefresh={refreshDetail} />
          )}

          {!isNew && effectiveId && (
            <LineItemsSection
              purchaseOrderId={effectiveId}
              lineItems={detail?.line_items ?? []}
              currencyCode={currencyCode}
              initialParsingJobId={initialParsingJobId}
              onAdd={addLineItem}
              onUpdate={updateLineItem}
              onRemove={removeLineItem}
              onRefresh={refreshDetail}
            />
          )}

          {!isNew && effectiveId && (
            <ReceivingSection
              purchaseOrderId={effectiveId}
              warehouseId={po?.warehouse_id ?? null}
              warehouses={warehouses}
              onAssignWarehouse={handleAssignWarehouse}
              lineItems={detail?.line_items ?? []}
            />
          )}
        </div>
      </div>
    </div>
  )
}

// ── Missing supplier review ──────────────────────────────────────────────────

interface MissingSupplierReviewProps {
  purchaseOrderId: string
  suggestedName: string
  suppliers: Supplier[]
  onAssigned: (supplierId: string) => void
}

// Mirrors the backend's FindByName lookup exactly, so what this shows is
// what submitting will actually do: lower-case, strip periods/commas
// (AI-extracted text and manually-typed names disagree on trailing
// punctuation for legal suffixes like "Lda."/"Lda"), collapse whitespace
// runs -- but leave letters (incl. accented ones) untouched.
function normalizeSupplierName(name: string): string {
  return name.toLowerCase().replace(/[.,]/g, '').replace(/\s+/g, ' ').trim()
}

function MissingSupplierReview({ purchaseOrderId, suggestedName, suppliers, onAssigned }: MissingSupplierReviewProps) {
  const { token } = useAuth()
  const [name, setName] = useState(suggestedName)
  const { t } = useTranslation('procurement')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const matchedSupplier = suppliers.find(s => normalizeSupplierName(s.name) === normalizeSupplierName(name))

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setSaving(true)
    try {
      const po = await procurementApi.createSupplierAndAssignToPurchaseOrder(token ?? '', purchaseOrderId, { name })
      if (po.supplier_id) onAssigned(po.supplier_id)
    } catch (err) {
      setError(apiErrorText(err) ?? t('missing.createFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="rounded-xl border dark:border-amber-500/30 border-amber-300 dark:bg-amber-500/5 bg-amber-50/40 p-4 space-y-2">
      <p className="flex items-center gap-1.5 text-xs font-semibold text-warning">
        <Sparkles size={12} /> {t('missing.title')}
      </p>
      <p className="text-2xs text-ink-3">
        {t('missing.hint')}
      </p>
      {suggestedName && (
        <p className="text-2xs text-accent">{t('missing.suggested')}</p>
      )}
      {matchedSupplier && (
        <p className="text-2xs text-success">
          {t('missing.matches', { name: matchedSupplier.name })}
        </p>
      )}
      <form onSubmit={handleCreate} className="flex items-center gap-2">
        <input
          aria-label={t('missing.newName')} required value={name} onChange={e => setName(e.target.value)}
          placeholder={t('missing.newName')}
          className="flex-1 px-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
        <button type="submit" disabled={saving || !name.trim()}
          className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 whitespace-nowrap">
          {saving ? (matchedSupplier ? t('missing.assigning') : t('missing.creating')) : (matchedSupplier ? t('missing.assignExisting') : t('missing.createAssign'))}
        </button>
      </form>
      {error && <p className="text-2xs text-danger">{error}</p>}
    </div>
  )
}
