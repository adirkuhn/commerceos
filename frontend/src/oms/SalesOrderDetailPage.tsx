import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft, Trash2, Printer, Edit3 } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { activityText } from './activityText'
import { useAuth } from '../auth/AuthContext'
import { useSalesOrderDetail } from './hooks/useSalesOrderDetail'
import { useSalesOrderActivities } from './hooks/useSalesOrderActivities'
import { useBolShipmentConfirmations } from './hooks/useBolShipmentConfirmations'
import type { SalesOrderStatus } from './types'
import { PickingListPrintView } from '../pages/print/PickingListPrintView'
import { InvoiceWithSeller } from '../pages/print/InvoiceWithSeller'
import * as shippingApi from '../shipping/api'
import { AddressConfirmModal } from '../shipping/AddressConfirmModal'
import { LabelPreviewModal } from '../shipping/LabelPreviewModal'
import { ExistingShipmentConfirmModal } from '../shipping/ExistingShipmentConfirmModal'
import { loadShipmentLabelIntoTab, openShipmentLabel } from '../shipping/openLabel'
import { LabelOpenedNotice } from '../shipping/LabelOpenedNotice'
import { useLabelConnection } from '../shipping/useLabelConnection'
import { LabelConnectionPicker } from '../shipping/ShippingProviderPicker'
import { providerLabel } from '../webhooks/providerLabel'
import type { RecipientAddress, Shipment } from '../shipping/types'
import { useConfirm } from '../components/common/useConfirm'
import { STATUS_OPTIONS, addressesEqual, customerName } from './salesOrderDisplay'
import { salesOrderStatusLabel } from './statusLabel'
import { OrderPaymentCard } from './OrderPaymentCard'
import { SalesOrderPaymentsCard } from './SalesOrderPaymentsCard'
import { ShipmentTrackingSection } from './ShipmentTrackingSection'
import { BolConfirmationCard } from './BolConfirmationCard'
import { SoldExpiryNote } from './SoldExpiryNote'
import { useDateFormat } from '../format/DateFormatProvider'
import { Money } from '../format/amountVisibility'
import { apiErrorText } from '../api/apiErrorText'

// ── Sales Order Detail Page ──────────────────────────────────────────────────

// Statuses at which a Bol order's shipment should have been sent to Bol --
// the Bol confirmation card shows from then on.
const BOL_SHIPPED_STATUSES: SalesOrderStatus[] = ['shipped', 'completed']

export function SalesOrderDetailPage({ salesOrderId, onClose }: { salesOrderId: string; onClose: () => void }) {
  const { formatDate, formatDateTime } = useDateFormat()
  const { t } = useTranslation(['orders', 'common'])
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { detail, loading, error, update, remove } = useSalesOrderDetail(token ?? '', salesOrderId)
  const { activities, refresh: refreshActivities } = useSalesOrderActivities(token ?? '', salesOrderId)
  const isBolOrder = detail?.sales_order.source === 'bol'
  const bolConfirmations = useBolShipmentConfirmations(token ?? '', salesOrderId, isBolOrder)
  const [savingStatus, setSavingStatus] = useState(false)
  const [actionError, setActionError] = useState('')
  const [deleting, setDeleting] = useState(false)
  const [editingNote, setEditingNote] = useState(false)
  const [noteDraft, setNoteDraft] = useState('')
  const [savingNote, setSavingNote] = useState(false)

  function startEditingNote() {
    setNoteDraft(detail?.sales_order.customer_note ?? '')
    setEditingNote(true)
  }

  async function saveNote() {
    setSavingNote(true)
    setActionError('')
    try {
      await update({ customer_note: noteDraft })
      setEditingNote(false)
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('detail.failedSaveNote'))
    } finally {
      setSavingNote(false)
    }
  }
  // Full-screen overlay in place of the normal detail chrome, same pattern
  // as this page's own list<->detail<->create switch -- this app has no
  // router, so a one-off view is just a conditional render, not a route.
  const [printMode, setPrintMode] = useState<'picking-list' | 'invoice' | null>(null)

  // "Create label" -- a second, independent entry point for creating a
  // shipment, alongside the one on the Packing page. Unlike Packing's
  // flow, a provider with its own order dashboard (MyParcel's Orders API,
  // ExportMyParcelOrderUseCase) gets the order exported first -- confirmed
  // address in hand from the same modal -- before the actual shipment.
  const shipping = useLabelConnection(token ?? '')
  const { connection: shippingConnection, labelSize, setLabelSize, autoPrintEnabled } = shipping
  const { hasCarrierChoice, packageTypeIds } = shipping.capabilities
  // Label creation is a 3-step flow: 'address' (AddressConfirmModal, no API
  // call) -> 'preview' (LabelPreviewModal, still no API call) -> Print
  // label actually creates the shipment (see handlePrintLabel). Same shape
  // as Picking.tsx's identical flow.
  const [labelStep, setLabelStep] = useState<'address' | 'preview' | null>(null)
  const [pendingRecipient, setPendingRecipient] = useState<RecipientAddress | null>(null)
  const [pendingCarrier, setPendingCarrier] = useState<number | null>(null)
  const [pendingPackageType, setPendingPackageType] = useState<number | null>(null)
  const [createdShipmentId, setCreatedShipmentId] = useState<string | null>(null)
  // See Picking.tsx's identical labelTab doc.
  const [labelTab, setLabelTab] = useState<Window | null>(null)
  // See Picking.tsx's identical existingShipment doc.
  const [existingShipment, setExistingShipment] = useState<Shipment | null>(null)
  // Every shipment ever created for this order -- normally one, but
  // nothing stops a second after a failed/lost first attempt (see
  // ShipmentRepository.FindBySalesOrderID's own doc comment). Each gets
  // its own tracking timeline below.
  const [shipments, setShipments] = useState<Shipment[]>([])
  const loadShipments = useCallback(() => {
    if (!token || !detail) return
    shippingApi.listShipmentsForOrder(token, detail.sales_order.id).then(setShipments).catch(() => setShipments([]))
  }, [token, detail])
  useEffect(() => { loadShipments() }, [loadShipments])
  // See Picking.tsx's identical handleCreateLabelClick doc.
  async function handleCreateLabelClick() {
    if (!token || !detail || !shippingConnection) return
    const shipments = await shippingApi.listShipmentsForOrder(token, detail.sales_order.id)
    // A cancelled shipment (wrong carrier, "Delete & create new") no longer
    // blocks creating a fresh one -- same rule CreateShipmentUseCase's own
    // duplicate-guard applies server-side.
    const active = shipments.find(s => !s.cancelled_at)
    if (active) {
      setExistingShipment(active)
      return
    }
    setLabelStep('address')
  }

  // See Picking.tsx's identical handleOpenLabel doc.
  async function handleOpenLabel(shipmentId: string) {
    if (!token) return
    try {
      setLabelTab(await openShipmentLabel(token, shipmentId, labelSize))
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('detail.failedLoadLabel'))
    }
  }

  // The actual shipment-creation call -- moved here from the old
  // AddressConfirmModal.onConfirm. Staff have already seen this exact data
  // laid out label-style in LabelPreviewModal and explicitly clicked "Print
  // label"; this is what that click does, then opens the label in a new
  // tab immediately (see Picking.tsx's identical handlePrintLabel doc on
  // why the blank tab opens synchronously, before either await).
  async function handlePrintLabel() {
    if (!token || !detail || !shippingConnection || !pendingRecipient) return
    const input = { carrier: shipping.carrierFor(pendingCarrier), recipient: pendingRecipient }
    const tab = window.open('', '_blank')
    let shipment
    try {
      if (shipping.capabilities.exportsOrders) {
        await shippingApi.exportMyParcelOrder(token, shippingConnection.id, detail.sales_order.id, input)
      }
      shipment = await shippingApi.createShipment(token, {
        ...input, package_type: pendingPackageType ?? shipping.defaultPackageType, connection_id: shippingConnection.id, sales_order_id: detail.sales_order.id,
      })
    } catch (err) {
      tab?.close()
      throw err
    }
    setLabelStep(null)
    setCreatedShipmentId(shipment.id)
    try {
      setLabelTab(await loadShipmentLabelIntoTab(tab, token, shipment.id, labelSize))
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('detail.failedLoadLabel'))
    }
    refreshActivities()
    loadShipments()
  }

  if (printMode && detail) {
    return printMode === 'picking-list' ? (
      <PickingListPrintView order={detail.sales_order} lineItems={detail.line_items} onClose={() => setPrintMode(null)} />
    ) : (
      <InvoiceWithSeller order={detail.sales_order} lineItems={detail.line_items} onClose={() => setPrintMode(null)} />
    )
  }

  async function handleStatusChange(status: SalesOrderStatus) {
    setSavingStatus(true)
    setActionError('')
    try {
      await update({ status })
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('detail.failedUpdateStatus'))
    } finally {
      setSavingStatus(false)
    }
  }

  async function handleDelete() {
    if (!detail) return
    if (!(await confirm({ title: t('detail.deleteTitle', { number: detail.sales_order.number || detail.sales_order.external_order_id }), message: t('detail.deleteMessage'), confirmLabel: t('detail.deleteConfirm'), destructive: true }))) return
    setDeleting(true)
    setActionError('')
    try {
      await remove()
      onClose()
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('detail.failedDelete'))
      setDeleting(false)
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> {t('detail.back')}
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight flex items-center gap-2">
            {detail ? `#${detail.sales_order.number || detail.sales_order.external_order_id}` : t('detail.title')}
            {detail?.sales_order.needs_stock_review && (
              <span className="px-2 py-0.5 rounded-full text-2xs font-medium bg-amber-500/15 text-warning">{t('detail.stockIssue')}</span>
            )}
            {detail?.sales_order.needs_sku_review && (
              <span className="px-2 py-0.5 rounded-full text-2xs font-medium bg-amber-500/15 text-warning">{t('detail.unmatchedSku')}</span>
            )}
          </h1>
        </div>
        {detail && (
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPrintMode('picking-list')}
              title={t('detail.printPickingList')}
              className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
            >
              <Printer size={13} /> {t('detail.pickingList')}
            </button>
            <button
              onClick={() => setPrintMode('invoice')}
              title={t('detail.printInvoice')}
              className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
            >
              <Printer size={13} /> {t('detail.invoice')}
            </button>
            {shippingConnection && !createdShipmentId && (
              <button
                onClick={handleCreateLabelClick}
                title={t('detail.createLabelTitle')}
                className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
              >
                {t('detail.createLabel')}
              </button>
            )}
            {createdShipmentId && (
              <button
                onClick={() => createdShipmentId && handleOpenLabel(createdShipmentId)}
                title={t('detail.printLabelTitle')}
                className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
              >
                {/* auto_print_enabled already opened the print view once on
                    creation -- this button is then for re-printing (paper
                    jam, dismissed dialog, etc), not the first print. */}
                <Printer size={13} /> {autoPrintEnabled ? t('detail.retryPrint') : t('detail.printLabel')}
              </button>
            )}
            <select
              value={detail.sales_order.status}
              disabled={savingStatus}
              onChange={e => handleStatusChange(e.target.value as SalesOrderStatus)}
              className={clsx(
                'px-2.5 py-1.5 rounded-lg text-xs font-medium border border-line outline-none focus-visible:ring-2 focus-visible:ring-focus/60 cursor-pointer disabled:opacity-50',
                'bg-raised text-ink',
              )}
            >
              {STATUS_OPTIONS.map(s => <option key={s} value={s}>{salesOrderStatusLabel(s)}</option>)}
            </select>
            <button aria-label={t('detail.delete')}
              onClick={handleDelete}
              disabled={deleting}
              title={t('detail.delete')}
              className="w-8 h-8 flex items-center justify-center rounded-lg border border-line hover:bg-hover disabled:opacity-50 transition-colors"
            >
              <Trash2 size={14} className="text-ink-3" />
            </button>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-5xl mx-auto p-6 space-y-5">
          {loading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
          {(error || actionError) && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error || actionError}</p>}

          {detail && (
            <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-5 items-start">
              {/* Main column */}
              <div className="space-y-5 min-w-0">
                <div className="rounded-xl border border-line p-4 space-y-3">
                  <h3 className="text-xs font-semibold text-ink">{t('detail.lineItems')}</h3>
                  <table className="w-full text-xs">
                    <thead>
                      <tr className="border-b border-line">
                        <th className="text-left py-1.5 font-medium text-ink-4">{t('common:fields.product')}</th>
                        <th className="text-right py-1.5 font-medium text-ink-4">{t('detail.qty')}</th>
                        <th className="text-right py-1.5 font-medium text-ink-4">{t('common:fields.price')}</th>
                        <th className="text-right py-1.5 font-medium text-ink-4">{t('common:fields.total')}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-line/60">
                      {detail.line_items.map(li => (
                        <tr key={li.id}>
                          <td className="py-1.5">
                            <p className="text-ink">{li.name}</p>
                            {li.sku_code && (
                              <p className="text-2xs text-ink-4">{li.sku_code}</p>
                            )}
                            {li.barcode && (
                              <p className="text-2xs text-ink-4">{li.barcode}</p>
                            )}
                            <SoldExpiryNote date={li.sold_expiry_date} />
                            {!li.sku_id && (
                              <p className="text-2xs text-warning">{t('detail.notLinked')}</p>
                            )}
                          </td>
                          <td className="py-1.5 text-right text-ink-2">{li.quantity}</td>
                          <td className="py-1.5 text-right text-ink-2">
                            <Money minor={li.price_minor_units} currency={detail.sales_order.currency_code} />
                          </td>
                          <td className="py-1.5 text-right font-medium text-ink">
                            <Money minor={li.total_minor_units} currency={detail.sales_order.currency_code} />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>

                  <div className="pt-2 border-t border-line space-y-1 text-xs">
                    <div className="flex justify-between text-ink-3">
                      <span>{t('detail.discount')}</span>
                      <span><Money minor={detail.sales_order.discount_total_minor_units} currency={detail.sales_order.currency_code} /></span>
                    </div>
                    <div className="flex justify-between text-ink-3">
                      <span>{t('detail.shipping')}</span>
                      <span><Money minor={detail.sales_order.shipping_total_minor_units} currency={detail.sales_order.currency_code} /></span>
                    </div>
                    <div className="flex justify-between text-ink-3">
                      <span>{t('detail.tax')}</span>
                      <span><Money minor={detail.sales_order.tax_total_minor_units} currency={detail.sales_order.currency_code} /></span>
                    </div>
                    <div className="flex justify-between font-semibold text-ink pt-1">
                      <span>{t('common:fields.total')}</span>
                      <span><Money minor={detail.sales_order.total_minor_units} currency={detail.sales_order.currency_code} /></span>
                    </div>
                  </div>
                </div>

                <OrderPaymentCard order={detail.sales_order} />

                <SalesOrderPaymentsCard payments={detail.payments ?? []} currency={detail.sales_order.currency_code} />

                {shipments.length > 0 && token && (
                  <div className="rounded-xl border border-line p-4 space-y-3 text-xs">
                    <p className="font-medium text-ink-2">{t('detail.shipments')}</p>
                    {shipments.map(s => <ShipmentTrackingSection key={s.id} token={token} shipment={s} onCancelled={loadShipments} />)}
                  </div>
                )}

                {isBolOrder && (BOL_SHIPPED_STATUSES.includes(detail.sales_order.status) || bolConfirmations.confirmations.length > 0) && (
                  <BolConfirmationCard confirmations={bolConfirmations.confirmations} error={bolConfirmations.error} />
                )}

                {activities.length > 0 && (
                  <div className="rounded-xl border border-line p-4 space-y-2 text-xs">
                    <p className="font-medium text-ink-2">{t('detail.timeline')}</p>
                    {activities.map(a => (
                      <div key={a.id} className="flex justify-between gap-3 text-ink-3">
                        <span className="text-ink">{activityText(t, a)}</span>
                        <span className="shrink-0">{formatDateTime(a.created_at)}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Aside column */}
              <div className="space-y-5 min-w-0">
                <div className="rounded-xl border border-line p-4 space-y-2 text-xs">
                  <div className="flex items-center justify-between">
                    <h3 className="text-xs font-semibold text-ink">{t('detail.notes')}</h3>
                    {!editingNote && (
                      <button
                        onClick={startEditingNote}
                        title={t('detail.editNotes')}
                        aria-label={t('detail.editNotes')}
                        className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      >
                        <Edit3 size={12} className="text-ink-3" />
                      </button>
                    )}
                  </div>
                  {editingNote ? (
                    <div className="space-y-2">
                      <label htmlFor="sales-order-note" className="sr-only">{t('detail.notes')}</label>
                      <textarea
                        id="sales-order-note"
                        value={noteDraft}
                        onChange={e => setNoteDraft(e.target.value)}
                        rows={3}
                        className="w-full px-2 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus"
                      />
                      <div className="flex items-center gap-2">
                        <button
                          onClick={saveNote}
                          disabled={savingNote}
                          className="px-2.5 py-1 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors disabled:opacity-50"
                        >
                          {t('detail.saveNote')}
                        </button>
                        <button
                          onClick={() => setEditingNote(false)}
                          disabled={savingNote}
                          className="px-2.5 py-1 rounded-lg border border-line text-xs font-medium text-ink-2 hover:bg-hover transition-colors disabled:opacity-50"
                        >
                          {t('common:actions.cancel')}
                        </button>
                      </div>
                    </div>
                  ) : detail.sales_order.customer_note ? (
                    <p className="text-ink whitespace-pre-wrap">{detail.sales_order.customer_note}</p>
                  ) : (
                    <p className="text-ink-4">{t('detail.noNotes')}</p>
                  )}
                </div>

                <div className="rounded-xl border border-line p-4 space-y-3 text-xs">
                  <h3 className="text-xs font-semibold text-ink">{t('detail.customer')}</h3>
                  <p>
                    {detail.sales_order.customer_id ? (
                      <Link to={`/customers/${detail.sales_order.customer_id}`} className="text-accent hover:underline">
                        {customerName(detail.sales_order)}
                      </Link>
                    ) : (
                      <span className="text-ink">{customerName(detail.sales_order)}</span>
                    )}
                  </p>

                  {detail.sales_order.billing?.email && (
                    <div>
                      <h4 className="text-2xs font-semibold caps-label text-ink-4 mb-1">{t('detail.contactInfo')}</h4>
                      <p className="text-ink">{detail.sales_order.billing.email}</p>
                    </div>
                  )}

                  {detail.sales_order.ship_by_date && (
                    <div>
                      <h4 className="text-2xs font-semibold caps-label text-ink-4 mb-1">{t('detail.shipBy')}</h4>
                      <p className="text-ink">{formatDate(detail.sales_order.ship_by_date)}</p>
                    </div>
                  )}

                  <div>
                    <h4 className="text-2xs font-semibold caps-label text-ink-4 mb-1">{t('detail.shippingAddress')}</h4>
                    <AddressLines address={detail.sales_order.shipping} />
                  </div>

                  <div>
                    <h4 className="text-2xs font-semibold caps-label text-ink-4 mb-1">{t('detail.billingAddress')}</h4>
                    {addressesEqual(detail.sales_order.billing, detail.sales_order.shipping) ? (
                      <p className="text-ink-4">{t('detail.sameAsShipping')}</p>
                    ) : (
                      <AddressLines address={detail.sales_order.billing} />
                    )}
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {labelStep === 'address' && detail && (
        <AddressConfirmModal
          token={token ?? ''}
          initialAddress={detail.sales_order.shipping}
          fallbackEmail={detail.sales_order.billing?.email}
          initialRecipient={pendingRecipient ?? undefined}
          initialCarrier={pendingCarrier ?? undefined}
          initialPackageType={pendingPackageType ?? undefined}
          defaultCarrier={shipping.defaultCarrier}
          defaultPackageType={shipping.defaultPackageType}
          onNext={({ carrier, packageType, recipient }) => { setPendingCarrier(carrier); setPendingPackageType(packageType); setPendingRecipient(recipient); setLabelStep('preview') }}
          onClose={() => setLabelStep(null)}
          providerPicker={<LabelConnectionPicker shipping={shipping} />}
          showCarrier={hasCarrierChoice}
          packageTypeIds={packageTypeIds}
        />
      )}
      {labelStep === 'preview' && pendingRecipient && (
        <LabelPreviewModal
          recipient={pendingRecipient}
          carrier={shipping.carrierFor(pendingCarrier)}
          packageType={pendingPackageType ?? shipping.defaultPackageType}
          providerName={shippingConnection ? providerLabel(shippingConnection.provider) : undefined}
          showCarrier={hasCarrierChoice}
          labelSize={labelSize}
          onLabelSizeChange={setLabelSize}
          onBack={() => setLabelStep('address')}
          onPrint={handlePrintLabel}
          onClose={() => setLabelStep(null)}
        />
      )}
      {labelTab && (
        <LabelOpenedNotice onPrint={() => labelTab.print()} onDismiss={() => setLabelTab(null)} />
      )}
      {existingShipment && token && (
        <ExistingShipmentConfirmModal
          token={token}
          shipment={existingShipment}
          onCancel={() => setExistingShipment(null)}
          onReprint={() => {
            setCreatedShipmentId(existingShipment.id)
            handleOpenLabel(existingShipment.id)
            setExistingShipment(null)
          }}
          onCreateNew={() => {
            setExistingShipment(null)
            setLabelStep('address')
          }}
        />
      )}
      {confirmDialog}
    </div>
  )
}

function AddressLines({ address }: { address?: { first_name?: string; last_name?: string; company?: string; address_1?: string; address_2?: string; city?: string; state?: string; postcode?: string; country?: string; email?: string; phone?: string } }) {
  if (!address) return <p className="text-xs text-ink-4">—</p>
  const name = [address.first_name, address.last_name].filter(Boolean).join(' ')
  const lines = [
    name,
    address.company,
    address.address_1,
    address.address_2,
    [address.city, address.state, address.postcode].filter(Boolean).join(', '),
    address.country,
    address.email,
    address.phone,
  ].filter(Boolean)

  if (lines.length === 0) return <p className="text-xs text-ink-4">—</p>

  return (
    <>
      {lines.map((line, i) => (
        <p key={i} className="text-xs text-ink-2">{line}</p>
      ))}
    </>
  )
}
