import { useState } from 'react'
import { Search, Filter, Plus, ArrowRightLeft, AlertTriangle, Calendar } from 'lucide-react'
import clsx from 'clsx'

interface StockItem {
  id: string
  product: string
  sku: string
  location: string
  qty: number
  reserved: number
  available: number
  lot: string
  expiry: string
  status: 'ok' | 'expiring' | 'expired' | 'low'
}

const STOCK: StockItem[] = [
  { id: '1', product: 'Wireless Headphones Pro', sku: 'WHP-001', location: 'Warehouse A — A01-B03', qty: 423, reserved: 12, available: 411, lot: 'LOT-2024-001', expiry: '—', status: 'ok' },
  { id: '2', product: 'USB-C Hub 7-in-1', sku: 'UCH-007', location: 'Warehouse A — A02-C01', qty: 187, reserved: 8, available: 179, lot: 'LOT-2024-002', expiry: '—', status: 'ok' },
  { id: '3', product: 'Organic Oat Cookies', sku: 'OOC-500', location: 'Warehouse B — C04-D02', qty: 340, reserved: 20, available: 320, lot: 'LOT-2026-012', expiry: '2026-05-22', status: 'expiring' },
  { id: '4', product: 'Vitamin C 1000mg', sku: 'VTC-1K', location: 'Warehouse B — D01-A01', qty: 120, reserved: 5, available: 115, lot: 'LOT-2024-089', expiry: '2026-04-30', status: 'expired' },
  { id: '5', product: 'Mouse Pad XL Gaming', sku: 'MPX-XL', location: 'Warehouse A — A03-B01', qty: 8, reserved: 2, available: 6, lot: 'LOT-2025-044', expiry: '—', status: 'low' },
  { id: '6', product: 'Webcam 4K Ultra', sku: 'WCM-4K', location: 'Warehouse A — B02-C03', qty: 56, reserved: 4, available: 52, lot: 'LOT-2025-033', expiry: '—', status: 'ok' },
  { id: '7', product: 'Laptop Stand Pro', sku: 'LSP-002', location: 'Warehouse A — A04-D01', qty: 312, reserved: 15, available: 297, lot: 'LOT-2025-077', expiry: '—', status: 'ok' },
  { id: '8', product: 'Protein Bar Chocolate', sku: 'PBC-BOX', location: 'Warehouse B — C03-B02', qty: 890, reserved: 45, available: 845, lot: 'LOT-2026-009', expiry: '2026-05-19', status: 'expiring' },
  { id: '9', product: 'LED Desk Lamp Smart', sku: 'LDL-SMT', location: 'Store Brussels', qty: 12, reserved: 0, available: 12, lot: 'LOT-2025-091', expiry: '—', status: 'low' },
  { id: '10', product: 'Cable Management Kit', sku: 'CMK-001', location: 'Warehouse A — A01-A04', qty: 890, reserved: 30, available: 860, lot: 'LOT-2025-055', expiry: '—', status: 'ok' },
]

const STATUS_STYLE: Record<string, string> = {
  ok: 'bg-green-500/15 text-green-400',
  expiring: 'bg-amber-500/15 text-amber-400',
  expired: 'bg-red-500/15 text-red-400',
  low: 'bg-orange-500/15 text-orange-400',
}

const STATUS_ICON: Record<string, React.ElementType> = {
  expiring: Calendar,
  expired: AlertTriangle,
  low: AlertTriangle,
}

const LOCATIONS = ['All Locations', 'Warehouse A', 'Warehouse B', 'Store Brussels']

export default function Inventory() {
  const [search, setSearch] = useState('')
  const [location, setLocation] = useState('All Locations')
  const [statusFilter, setStatusFilter] = useState('all')

  const filtered = STOCK.filter(s => {
    const matchSearch = s.product.toLowerCase().includes(search.toLowerCase()) || s.sku.toLowerCase().includes(search.toLowerCase())
    const matchLoc = location === 'All Locations' || s.location.includes(location)
    const matchStatus = statusFilter === 'all' || s.status === statusFilter
    return matchSearch && matchLoc && matchStatus
  })

  const counts = {
    ok: STOCK.filter(s => s.status === 'ok').length,
    expiring: STOCK.filter(s => s.status === 'expiring').length,
    expired: STOCK.filter(s => s.status === 'expired').length,
    low: STOCK.filter(s => s.status === 'low').length,
  }

  return (
    <div className="flex flex-col h-full">
      {/* Summary cards */}
      <div className="grid grid-cols-4 gap-3 p-4 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
        {([
          ['ok', 'OK Stock', counts.ok, 'green'],
          ['expiring', 'Expiring Soon', counts.expiring, 'amber'],
          ['expired', 'Expired', counts.expired, 'red'],
          ['low', 'Low Stock', counts.low, 'orange'],
        ] as const).map(([s, label, count, color]) => (
          <button
            key={s}
            onClick={() => setStatusFilter(statusFilter === s ? 'all' : s)}
            className={clsx(
              'text-left p-3 rounded-xl border transition-all',
              statusFilter === s
                ? `dark:bg-${color}-500/10 bg-${color}-50 dark:border-${color}-500/30 border-${color}-200`
                : 'dark:bg-[#1F2937] bg-white dark:border-[#2D3748] border-gray-200 hover:dark:bg-[#2D3748]/50 hover:bg-gray-50',
            )}
          >
            <p className="text-xl font-semibold dark:text-gray-100 text-gray-900">{count}</p>
            <p className="text-[10px] dark:text-gray-400 text-gray-500 mt-0.5">{label}</p>
          </button>
        ))}
      </div>

      {/* Toolbar */}
      <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search product, SKU..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        <select
          value={location}
          onChange={e => setLocation(e.target.value)}
          className="px-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-600 outline-none"
        >
          {LOCATIONS.map(l => <option key={l}>{l}</option>)}
        </select>

        <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
          <Filter size={13} /> Filters
        </button>

        <div className="flex items-center gap-2 ml-auto">
          <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
            <ArrowRightLeft size={13} /> Transfer
          </button>
          <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 font-medium">
            <Plus size={13} /> Adjust Stock
          </button>
        </div>
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-4 py-2.5 w-8">
                <input type="checkbox" className="rounded accent-blue-600 w-3 h-3" />
              </th>
              {['Product', 'SKU', 'Location', 'Qty', 'Reserved', 'Available', 'Lot', 'Expiry', 'Status'].map(h => (
                <th key={h} className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider whitespace-nowrap">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {filtered.map(s => {
              const StatusIconComp = STATUS_ICON[s.status]
              return (
                <tr key={s.id} className="hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors">
                  <td className="px-4 py-2.5">
                    <input type="checkbox" className="rounded accent-blue-600 w-3 h-3" />
                  </td>
                  <td className="px-3 py-2.5">
                    <p className="font-medium dark:text-gray-100 text-gray-900">{s.product}</p>
                  </td>
                  <td className="px-3 py-2.5">
                    <span className="font-mono dark:text-gray-400 text-gray-500">{s.sku}</span>
                  </td>
                  <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500 text-[10px]">{s.location}</td>
                  <td className="px-3 py-2.5 dark:text-gray-200 text-gray-800 font-medium tabular-nums">{s.qty.toLocaleString()}</td>
                  <td className="px-3 py-2.5 text-amber-400 tabular-nums">{s.reserved}</td>
                  <td className="px-3 py-2.5 text-green-400 font-medium tabular-nums">{s.available.toLocaleString()}</td>
                  <td className="px-3 py-2.5 font-mono dark:text-gray-500 text-gray-400 text-[10px]">{s.lot}</td>
                  <td className="px-3 py-2.5">
                    {s.expiry === '—' ? (
                      <span className="dark:text-gray-600 text-gray-300">—</span>
                    ) : (
                      <span className={clsx(
                        'font-mono text-[10px]',
                        s.status === 'expired' ? 'text-red-400' : 'text-amber-400',
                      )}>
                        {s.expiry}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2.5">
                    <span className={clsx('flex items-center gap-1 w-fit px-2 py-0.5 rounded-full text-[10px] font-medium capitalize', STATUS_STYLE[s.status])}>
                      {StatusIconComp && <StatusIconComp size={10} />}
                      {s.status}
                    </span>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}
