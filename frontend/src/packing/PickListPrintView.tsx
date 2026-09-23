import { Printer, X } from 'lucide-react'
import { usePickList } from '../oms/hooks/usePickList'

interface PickListPrintViewProps {
  token: string
  orderIds: string[]
  onClose: () => void
}

// Multi-order pick list: what to fetch off the shelves for the selected
// orders, one row per SKU, before scanning them item by item in packing.
// Read-only -- no picking session is created. Same plain-HTML print sheet
// as PickingListPrintView (white paper, print: variants hide the chrome),
// but printing is on demand since the data loads asynchronously.
export function PickListPrintView({ token, orderIds, onClose }: PickListPrintViewProps) {
  const { pickList, loading, error } = usePickList(token, orderIds)
  const items = pickList?.items ?? []
  const totalItems = items.reduce((sum, it) => sum + it.total_quantity, 0)

  return (
    <div className="fixed inset-0 z-50 bg-white overflow-y-auto">
      <div className="print:hidden fixed top-4 right-4 flex gap-2">
        <button
          onClick={() => window.print()}
          disabled={!pickList}
          className="h-9 px-3 flex items-center gap-1.5 rounded-lg bg-gray-900 text-white text-sm disabled:opacity-40"
        >
          <Printer size={14} /> Print
        </button>
        <button aria-label="Close" title="Close"
          onClick={onClose}
          className="w-9 h-9 flex items-center justify-center rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-600"
        >
          <X size={16} />
        </button>
      </div>
      <div className="max-w-3xl mx-auto p-10 text-gray-900">
        <h1 className="text-2xl font-bold mb-1">Pick list</h1>
        <p className="text-sm text-gray-600 mb-6">
          {orderIds.length} {orderIds.length === 1 ? 'order' : 'orders'} · {totalItems} {totalItems === 1 ? 'item' : 'items'}
        </p>

        {loading && <p className="text-sm text-gray-500">Loading…</p>}
        {error && <p role="alert" className="text-sm text-red-700">{error}</p>}
        {pickList && items.length === 0 && <p className="text-sm text-gray-500">Nothing left to pick for these orders.</p>}

        {items.length > 0 && (
          <table className="w-full text-sm border-collapse">
            <thead>
              <tr className="border-b-2 border-gray-900">
                <th className="text-left py-2 w-8"><span className="sr-only">Picked</span></th>
                <th className="text-left py-2">Product</th>
                <th className="text-left py-2">SKU</th>
                <th className="text-right py-2">Qty</th>
                <th className="text-left py-2 pl-4">Orders</th>
              </tr>
            </thead>
            <tbody>
              {items.map(item => (
                <tr key={item.sku_id ?? `name:${item.name}`} className="border-b border-gray-300 align-top break-inside-avoid">
                  <td className="py-2.5"><span className="inline-block w-4 h-4 border border-gray-500" /></td>
                  <td className="py-2.5">
                    <p>{item.name}</p>
                    {item.barcode && <p className="font-mono text-xs text-gray-500">{item.barcode}</p>}
                  </td>
                  <td className="py-2.5 font-mono text-xs">
                    {item.sku_code || (item.sku_id ? '—' : <span className="font-sans text-amber-700">No SKU</span>)}
                  </td>
                  <td className="py-2.5 text-right text-base font-semibold">{item.total_quantity}</td>
                  <td className="py-2.5 pl-4 text-xs text-gray-600">
                    {item.orders.map(o => <p key={o.line_item_id}>#{o.order_number} × {o.quantity}</p>)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
