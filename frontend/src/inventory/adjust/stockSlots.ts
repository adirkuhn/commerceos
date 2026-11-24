import type { Batch } from '../types'

// One place stock can be adjusted: a warehouse's stock without a lot (the
// "general ledger", batchId '') or one lot in it.
export interface StockSlot {
  key: string
  warehouseId: string
  warehouseName: string
  batchId: string
  lot: string
  expiryDate: string | null
  quantity: number
  isLedger: boolean
  // A lot with nothing left (the no-lot slot is never empty: stock can
  // always be booked there).
  empty: boolean
}

export const slotKey = (warehouseId: string, batchId: string) => `${warehouseId}:${batchId}`

// Every slot, grouped by warehouse (the given order, then any warehouse only
// a lot still names): the no-lot slot first, then lots soonest-expiry first
// (first expired, first out), undated lots last.
export function stockSlots(warehouses: { id: string; name: string }[], batches: Batch[]): StockSlot[] {
  const ids = [...warehouses.map(w => w.id), ...batches.map(b => b.warehouse_id)]
  const order = [...new Set(ids)]
  const nameOf = (id: string) => warehouses.find(w => w.id === id)?.name ?? id
  const known = new Set(warehouses.map(w => w.id))

  return order.flatMap(warehouseId => {
    const here = batches.filter(b => b.warehouse_id === warehouseId)
    const ledger = here.find(b => b.is_general_ledger)
    const lots = here
      .filter(b => !b.is_general_ledger)
      .sort((a, b) => (a.expiry_date ?? '9999').localeCompare(b.expiry_date ?? '9999'))
    const slots: StockSlot[] = lots.map(b => ({
      key: slotKey(warehouseId, b.id), warehouseId, warehouseName: nameOf(warehouseId), batchId: b.id,
      lot: b.lot_number, expiryDate: b.expiry_date ?? null, quantity: b.quantity, isLedger: false, empty: b.quantity === 0,
    }))
    if (ledger || known.has(warehouseId)) {
      slots.unshift({
        key: slotKey(warehouseId, ''), warehouseId, warehouseName: nameOf(warehouseId), batchId: '',
        lot: '', expiryDate: null, quantity: ledger?.quantity ?? 0, isLedger: true, empty: false,
      })
    }
    return slots
  })
}
