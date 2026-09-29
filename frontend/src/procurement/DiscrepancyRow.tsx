import { useState } from 'react'
import { Check, PackageCheck, Link2 } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import * as procurementApi from './api'
import * as inventoryApi from '../inventory/api'
import type { PurchaseOrderLineItem, ReceivingDiscrepancy } from './types'
import type { Warehouse } from '../inventory/types'
import { discrepancyLabel } from './procurementHelpers'
import { useDateFormat } from '../format/DateFormatProvider'

interface DiscrepancyRowProps {
  discrepancy: ReceivingDiscrepancy
  allDiscrepancies: ReceivingDiscrepancy[]
  lineItem: PurchaseOrderLineItem | undefined
  warehouseId: string | null
  warehouses: Warehouse[]
  onAssignWarehouse: (warehouseId: string) => Promise<void>
  invoiceFileName: string
  disabled: boolean
  onReload: () => Promise<void> | void
}

export function DiscrepancyRow({
  discrepancy, allDiscrepancies, lineItem, warehouseId, warehouses, onAssignWarehouse, invoiceFileName, disabled, onReload,
}: DiscrepancyRowProps) {
  const { token } = useAuth()
  const { formatDate } = useDateFormat()
  const { t } = useTranslation(['procurement', 'common'])
  const [formOpen, setFormOpen] = useState(false)
  const [linkOpen, setLinkOpen] = useState(false)
  const [linkTargetId, setLinkTargetId] = useState('')
  const [acting, setActing] = useState(false)
  const [actionError, setActionError] = useState('')
  const canReceive = !disabled && !formOpen && lineItem?.sku_id && discrepancy.received_qty > 0
  const isAcknowledged = discrepancy.status === 'acknowledged'

  // Candidates for manual linking: sibling discrepancies of the opposite
  // missing/unexpected type in the same record that are still open -- the
  // matcher's own fuzzy pass already handled everything it could connect
  // confidently, this is the human escape hatch for what's left.
  const linkCandidates = !isAcknowledged && (discrepancy.type === 'unexpected_item' || discrepancy.type === 'missing_item')
    ? allDiscrepancies.filter(d => d.status !== 'acknowledged' && d.type === (discrepancy.type === 'unexpected_item' ? 'missing_item' : 'unexpected_item'))
    : []

  async function handleAcknowledge() {
    setActing(true)
    setActionError('')
    try {
      await procurementApi.acknowledgeReceivingDiscrepancy(token ?? '', discrepancy.id)
      await onReload()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : t('disc.ackFailed'))
    } finally {
      setActing(false)
    }
  }

  async function handleLink() {
    if (!linkTargetId) return
    setActing(true)
    setActionError('')
    try {
      const [unexpectedId, missingId] = discrepancy.type === 'unexpected_item'
        ? [discrepancy.id, linkTargetId]
        : [linkTargetId, discrepancy.id]
      await procurementApi.linkReceivingDiscrepancies(token ?? '', unexpectedId, missingId)
      setLinkOpen(false)
      await onReload()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : t('disc.linkFailed'))
    } finally {
      setActing(false)
    }
  }

  return (
    <div className={clsx(
      'rounded-lg border p-2.5 space-y-1.5',
      isAcknowledged
        ? 'dark:border-white/10 border-gray-200 dark:bg-white/[0.02] bg-gray-50'
        : 'dark:border-amber-500/30 border-amber-300 dark:bg-amber-500/5 bg-amber-50/40',
    )}>
      <div className="flex items-center justify-between gap-2">
        <span className={clsx('text-2xs font-semibold', isAcknowledged ? 'text-ink-4' : 'text-warning')}>
          {discrepancyLabel(discrepancy.type)}
        </span>
        <span className="text-2xs text-ink-4">
          {t('disc.quantities', { expected: discrepancy.expected_qty, invoice: discrepancy.received_qty })}
        </span>
      </div>
      <p className="text-2xs text-ink-2">
        {discrepancy.details}
        {discrepancy.expiry_date && t('disc.expires', { date: formatDate(discrepancy.expiry_date) })}
      </p>
      {isAcknowledged ? (
        <span className="flex items-center gap-1 text-2xs font-medium text-success">
          <Check size={10} /> {t('disc.acknowledged')}
        </span>
      ) : (
        <div className="flex flex-wrap items-center gap-3">
          {canReceive && (
            <button
              type="button" onClick={() => setFormOpen(true)}
              className="flex items-center gap-1 text-2xs font-medium text-accent hover:underline"
            >
              <PackageCheck size={10} /> {t('disc.receiveStock')}
            </button>
          )}
          {!disabled && (
            <button
              type="button" onClick={handleAcknowledge} disabled={acting}
              className="flex items-center gap-1 text-2xs font-medium text-ink-3 hover:underline disabled:opacity-50"
            >
              <Check size={10} /> {t('disc.markValid')}
            </button>
          )}
          {!disabled && linkCandidates.length > 0 && !linkOpen && (
            <button
              type="button" onClick={() => setLinkOpen(true)}
              className="flex items-center gap-1 text-2xs font-medium text-ink-3 hover:underline"
            >
              <Link2 size={10} /> {t('disc.linkTo')}
            </button>
          )}
        </div>
      )}
      {linkOpen && (
        <div className="flex items-center gap-2 pt-1">
          <select
            aria-label={t('disc.linkAria')}
            value={linkTargetId} onChange={e => setLinkTargetId(e.target.value)}
            className="flex-1 text-2xs rounded-md border border-line dark:bg-night-950 bg-white text-ink-2 px-2 py-1"
          >
            <option value="">{t('disc.selectMatching')}</option>
            {linkCandidates.map(c => <option key={c.id} value={c.id}>{c.details}</option>)}
          </select>
          <button
            type="button" onClick={handleLink} disabled={acting || !linkTargetId}
            className="text-2xs font-medium text-accent hover:underline disabled:opacity-50"
          >
            {t('disc.confirm')}
          </button>
          <button type="button" onClick={() => setLinkOpen(false)} className="text-2xs text-ink-4 hover:underline">
            {t('common:actions.cancel')}
          </button>
        </div>
      )}
      {actionError && <p className="text-2xs text-danger">{actionError}</p>}
      {!disabled && lineItem && !lineItem.sku_id && discrepancy.received_qty > 0 && (
        <p className="text-2xs text-ink-4">{t('disc.linkFirst')}</p>
      )}
      {formOpen && lineItem?.sku_id && (
        <ReceiveDiscrepancyForm
          skuId={lineItem.sku_id}
          warehouseId={warehouseId}
          warehouses={warehouses}
          onAssignWarehouse={onAssignWarehouse}
          defaultQuantity={discrepancy.received_qty}
          unitsPerPurchasedItem={lineItem.units_per_purchased_item}
          reason={t('disc.receivingReason', { file: invoiceFileName })}
          onDone={() => setFormOpen(false)}
        />
      )}
    </div>
  )
}

interface FlagDamagedItemFormProps {
  recordId: string
  lineItems: PurchaseOrderLineItem[]
  onDone: () => void
  onCancel: () => void
}

export function FlagDamagedItemForm({ recordId, lineItems, onDone, onCancel }: FlagDamagedItemFormProps) {
  const { token } = useAuth()
  const [lineItemId, setLineItemId] = useState(lineItems[0]?.id ?? '')
  const [qty, setQty] = useState('1')
  const [note, setNote] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const { t } = useTranslation(['procurement', 'common'])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const damagedQty = Number(qty)
    if (!lineItemId || !Number.isFinite(damagedQty) || damagedQty <= 0) {
      setError(t('damaged.pickLine'))
      return
    }
    setSaving(true)
    setError('')
    try {
      await procurementApi.flagDamagedLineItem(token ?? '', recordId, lineItemId, damagedQty, note)
      onDone()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('damaged.failed'))
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="rounded-lg border border-line p-2.5 space-y-2">
      <div className="flex items-center gap-2">
        <select
          value={lineItemId} onChange={e => setLineItemId(e.target.value)} required
          className="flex-1 text-2xs rounded-md border border-line dark:bg-night-950 bg-white text-ink-2 px-2 py-1"
        >
          {lineItems.map(li => <option key={li.id} value={li.id}>{li.raw_description}</option>)}
        </select>
        <input
          type="number" min="1" value={qty} onChange={e => setQty(e.target.value)} required
          className="w-16 text-2xs rounded-md border border-line dark:bg-night-950 bg-white text-ink-2 px-2 py-1"
        />
      </div>
      <input
        type="text" placeholder={t('damaged.note')} value={note} onChange={e => setNote(e.target.value)}
        className="w-full text-2xs rounded-md border border-line dark:bg-night-950 bg-white text-ink-2 px-2 py-1"
      />
      {error && <p className="text-2xs text-danger">{error}</p>}
      <div className="flex items-center gap-3">
        <button
          type="submit" disabled={saving}
          className="text-2xs font-medium text-accent hover:underline disabled:opacity-50"
        >
          {saving ? t('common:actions.saving') : t('damaged.flag')}
        </button>
        <button type="button" onClick={onCancel} className="text-2xs text-ink-4 hover:underline">
          {t('common:actions.cancel')}
        </button>
      </div>
    </form>
  )
}

interface ReceiveDiscrepancyFormProps {
  skuId: string
  warehouseId: string | null
  warehouses: Warehouse[]
  onAssignWarehouse: (warehouseId: string) => Promise<void>
  defaultQuantity: number
  // Same convention as PurchaseOrderLineItem.UnitsPerPurchasedItem: how many
  // stock units one purchased item (box/case) represents. defaultQuantity
  // and the "Qty" field below stay in purchased-item units throughout --
  // matching received_qty/expected_qty (see diffInvoiceAgainstPurchaseOrder,
  // which diffs the invoice's own quantity against the PO line's Quantity,
  // both already in that unit) -- only the actual receive-stock call is
  // scaled up to stock units, same split as ScanBookingItemUseCase's
  // scaleByLineItem on the scanner side.
  unitsPerPurchasedItem: number
  reason: string
  onDone: () => void
}

function ReceiveDiscrepancyForm({ skuId, warehouseId, warehouses, onAssignWarehouse, defaultQuantity, unitsPerPurchasedItem, reason, onDone }: ReceiveDiscrepancyFormProps) {
  const { token } = useAuth()
  const [quantity, setQuantity] = useState(String(defaultQuantity))
  const [lotNumber, setLotNumber] = useState('')
  const [expiryDate, setExpiryDate] = useState('')
  // Only used when the PO has no warehouse yet -- picking one here assigns
  // it to the PO for good (see onAssignWarehouse), it's not a per-scan choice.
  const [pickedWarehouseId, setPickedWarehouseId] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const { t } = useTranslation(['procurement', 'common'])
  const [done, setDone] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setSaving(true)
    try {
      let effectiveWarehouseId = warehouseId
      if (!effectiveWarehouseId) {
        if (!pickedWarehouseId) throw new Error(t('receive.pickWarehouse'))
        await onAssignWarehouse(pickedWarehouseId)
        effectiveWarehouseId = pickedWarehouseId
      }
      const effectiveMultiplier = unitsPerPurchasedItem > 0 ? unitsPerPurchasedItem : 1
      await inventoryApi.receiveStock(token ?? '', skuId, {
        warehouse_id: effectiveWarehouseId, quantity: parseInt(quantity, 10) * effectiveMultiplier,
        lot_number: lotNumber || undefined, expiry_date: expiryDate || null, reason,
      })
      setDone(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('receive.failed'))
    } finally {
      setSaving(false)
    }
  }

  if (done) {
    return <p className="text-2xs font-medium text-green-400">{t('receive.done')}</p>
  }

  const inputClass = 'w-full px-2 py-1 text-2xs rounded dark:bg-night-950 bg-white border dark:border-night-700 text-ink outline-none focus:ring-1 focus:ring-focus'

  return (
    <form onSubmit={handleSubmit} className="flex items-end gap-2 pt-1 flex-wrap">
      {!warehouseId && (
        <div>
          <label htmlFor={`receive-warehouse-${skuId}`} className="block text-2xs text-ink-4 mb-0.5">{t('receive.warehouse')}</label>
          <select
            id={`receive-warehouse-${skuId}`} required value={pickedWarehouseId}
            onChange={e => setPickedWarehouseId(e.target.value)} className={clsx(inputClass, 'w-28')}
          >
            <option value="">{t('receive.select')}</option>
            {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
          </select>
        </div>
      )}
      <div>
        <label htmlFor={`receive-qty-${skuId}`} className="block text-2xs text-ink-4 mb-0.5">{t('receive.qty')}</label>
        <input id={`receive-qty-${skuId}`} type="number" min="1" required value={quantity} onChange={e => setQuantity(e.target.value)} className={clsx(inputClass, 'w-14')} />
        {unitsPerPurchasedItem > 1 && Number.isFinite(parseInt(quantity, 10)) && (
          <p className="text-2xs text-ink-4 mt-0.5">{t('receive.units', { units: parseInt(quantity, 10) * unitsPerPurchasedItem })}</p>
        )}
      </div>
      <div>
        <label htmlFor={`receive-lot-${skuId}`} className="block text-2xs text-ink-4 mb-0.5">{t('receive.lot')}</label>
        <input id={`receive-lot-${skuId}`} value={lotNumber} onChange={e => setLotNumber(e.target.value)} className={clsx(inputClass, 'w-20')} />
      </div>
      <div>
        <label htmlFor={`receive-expiry-${skuId}`} className="block text-2xs text-ink-4 mb-0.5">{t('receive.expiry')}</label>
        <input id={`receive-expiry-${skuId}`} type="date" value={expiryDate} onChange={e => setExpiryDate(e.target.value)} className={inputClass} />
      </div>
      <button type="submit" disabled={saving} className="px-2.5 py-1 rounded-lg bg-accent-solid text-white text-2xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50">
        {saving ? t('common:actions.saving') : t('receive.save')}
      </button>
      <button type="button" onClick={onDone} className="px-2 py-1 text-2xs text-ink-4 hover:underline">
        {t('common:actions.cancel')}
      </button>
      {error && <p className="text-2xs text-red-400">{error}</p>}
    </form>
  )
}
