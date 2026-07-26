import { useState } from 'react'
import {
  Plus, Download, Upload, Search, Filter, MoreHorizontal,
  ChevronUp, ChevronDown, Package, Copy, Trash2, Edit3,
} from 'lucide-react'
import clsx from 'clsx'

interface Product {
  id: string
  name: string
  sku: string
  barcode: string
  category: string
  supplier: string
  price: number
  cost: number
  margin: number
  status: 'active' | 'inactive' | 'draft'
  stock: number
}

const MOCK: Product[] = [
  { id: '1', name: 'Wireless Headphones Pro', sku: 'WHP-001', barcode: '8714574000282', category: 'Electronics', supplier: 'TechWorld BV', price: 249.99, cost: 89.50, margin: 64.2, status: 'active', stock: 423 },
  { id: '2', name: 'USB-C Hub 7-in-1', sku: 'UCH-007', barcode: '8714574000299', category: 'Accessories', supplier: 'ConnectPro Ltd', price: 99.99, cost: 32.00, margin: 68.0, status: 'active', stock: 187 },
  { id: '3', name: 'Mechanical Keyboard TKL', sku: 'MKT-TKL', barcode: '8714574000305', category: 'Peripherals', supplier: 'KeyMaster GmbH', price: 299.99, cost: 124.00, margin: 58.7, status: 'active', stock: 94 },
  { id: '4', name: 'Webcam 4K Ultra', sku: 'WCM-4K', barcode: '8714574000312', category: 'Electronics', supplier: 'TechWorld BV', price: 299.99, cost: 98.00, margin: 67.3, status: 'active', stock: 56 },
  { id: '5', name: 'Laptop Stand Pro', sku: 'LSP-002', barcode: '8714574000329', category: 'Accessories', supplier: 'ErgoCo NV', price: 99.99, cost: 28.50, margin: 71.5, status: 'active', stock: 312 },
  { id: '6', name: 'Mouse Pad XL Gaming', sku: 'MPX-XL', barcode: '8714574000336', category: 'Accessories', supplier: 'GameGear BV', price: 39.99, cost: 9.80, margin: 75.5, status: 'active', stock: 8 },
  { id: '7', name: 'HDMI Cable 2.1 2m', sku: 'HDM-2M', barcode: '8714574000343', category: 'Cables', supplier: 'ConnectPro Ltd', price: 24.99, cost: 6.20, margin: 75.2, status: 'inactive', stock: 0 },
  { id: '8', name: 'Portable SSD 1TB', sku: 'SSD-1TB', barcode: '8714574000350', category: 'Storage', supplier: 'DataStore AG', price: 149.99, cost: 72.00, margin: 52.0, status: 'draft', stock: 0 },
  { id: '9', name: 'LED Desk Lamp Smart', sku: 'LDL-SMT', barcode: '8714574000367', category: 'Lighting', supplier: 'LightTech BV', price: 79.99, cost: 24.00, margin: 70.0, status: 'active', stock: 145 },
  { id: '10', name: 'Cable Management Kit', sku: 'CMK-001', barcode: '8714574000374', category: 'Accessories', supplier: 'ErgoCo NV', price: 29.99, cost: 7.50, margin: 75.0, status: 'active', stock: 890 },
]

const STATUS_STYLES = {
  active: 'bg-green-500/15 text-green-400',
  inactive: 'bg-gray-500/15 text-gray-400',
  draft: 'bg-amber-500/15 text-amber-400',
}

const STOCK_COLOR = (n: number) =>
  n === 0 ? 'text-red-400' : n < 15 ? 'text-amber-400' : 'dark:text-gray-200 text-gray-800'

type SortKey = keyof Product
type SortDir = 'asc' | 'desc'

export default function Products() {
  const [search, setSearch] = useState('')
  const [sortKey, setSortKey] = useState<SortKey>('name')
  const [sortDir, setSortDir] = useState<SortDir>('asc')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [openMenu, setOpenMenu] = useState<string | null>(null)

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) setSortDir(d => d === 'asc' ? 'desc' : 'asc')
    else { setSortKey(key); setSortDir('asc') }
  }

  const filtered = MOCK
    .filter(p =>
      p.name.toLowerCase().includes(search.toLowerCase()) ||
      p.sku.toLowerCase().includes(search.toLowerCase()) ||
      p.barcode.includes(search)
    )
    .sort((a, b) => {
      const av = a[sortKey], bv = b[sortKey]
      const cmp = typeof av === 'string'
        ? av.localeCompare(bv as string)
        : (av as number) - (bv as number)
      return sortDir === 'asc' ? cmp : -cmp
    })

  const toggleSelect = (id: string) =>
    setSelected(prev => {
      const s = new Set(prev)
      s.has(id) ? s.delete(id) : s.add(id)
      return s
    })

  const allSelected = filtered.length > 0 && filtered.every(p => selected.has(p.id))
  const toggleAll = () =>
    setSelected(allSelected ? new Set() : new Set(filtered.map(p => p.id)))

  function SortIcon({ k }: { k: SortKey }) {
    if (sortKey !== k) return <ChevronUp size={12} className="opacity-20" />
    return sortDir === 'asc' ? <ChevronUp size={12} /> : <ChevronDown size={12} />
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search by name, SKU, barcode..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white dark:border-[#2D3748] border-gray-200 border dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100 transition-colors">
          <Filter size={13} /> Filters
        </button>

        {selected.size > 0 && (
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg dark:bg-blue-600/20 bg-blue-50 border dark:border-blue-500/30 border-blue-200">
            <span className="text-xs text-blue-400 font-medium">{selected.size} selected</span>
            <button className="text-xs text-red-400 hover:text-red-300 flex items-center gap-1">
              <Trash2 size={11} /> Delete
            </button>
          </div>
        )}

        <div className="flex items-center gap-2 ml-auto">
          <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100 transition-colors">
            <Upload size={13} /> Import
          </button>
          <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100 transition-colors">
            <Download size={13} /> Export
          </button>
          <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 transition-colors font-medium">
            <Plus size={13} /> New Product
          </button>
        </div>
      </div>

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">{filtered.length} products</span>
        <span className="text-xs dark:text-gray-500 text-gray-400">·</span>
        <span className="text-xs dark:text-gray-500 text-gray-400">
          <span className="text-green-400 font-medium">{MOCK.filter(p => p.status === 'active').length}</span> active
        </span>
        <span className="text-xs dark:text-gray-500 text-gray-400">·</span>
        <span className="text-xs dark:text-gray-500 text-gray-400">
          <span className="text-red-400 font-medium">{MOCK.filter(p => p.stock < 10).length}</span> low stock
        </span>
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-4 py-2.5 w-8">
                <input
                  type="checkbox"
                  checked={allSelected}
                  onChange={toggleAll}
                  className="rounded accent-blue-600 w-3 h-3"
                />
              </th>
              <th className="px-2 py-2.5 w-8" />
              {([
                ['name', 'Product'],
                ['sku', 'SKU'],
                ['barcode', 'Barcode'],
                ['category', 'Category'],
                ['supplier', 'Supplier'],
                ['price', 'Price'],
                ['cost', 'Cost'],
                ['margin', 'Margin'],
                ['stock', 'Stock'],
                ['status', 'Status'],
              ] as [SortKey, string][]).map(([k, label]) => (
                <th
                  key={k}
                  onClick={() => toggleSort(k)}
                  className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider cursor-pointer hover:dark:text-gray-300 hover:text-gray-600 transition-colors select-none whitespace-nowrap"
                >
                  <span className="flex items-center gap-1">
                    {label} <SortIcon k={k} />
                  </span>
                </th>
              ))}
              <th className="px-3 py-2.5 w-8" />
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {filtered.map(p => (
              <tr
                key={p.id}
                className={clsx(
                  'group hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors',
                  selected.has(p.id) && 'dark:bg-blue-600/5 bg-blue-50',
                )}
              >
                <td className="px-4 py-2.5">
                  <input
                    type="checkbox"
                    checked={selected.has(p.id)}
                    onChange={() => toggleSelect(p.id)}
                    className="rounded accent-blue-600 w-3 h-3"
                  />
                </td>
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-[#2D3748] bg-gray-200 flex items-center justify-center">
                    <Package size={12} className="dark:text-gray-500 text-gray-400" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[180px]">
                  <p className="font-medium dark:text-gray-100 text-gray-900 truncate">{p.name}</p>
                </td>
                <td className="px-3 py-2.5">
                  <span className="font-mono dark:text-gray-400 text-gray-500">{p.sku}</span>
                </td>
                <td className="px-3 py-2.5">
                  <span className="font-mono dark:text-gray-500 text-gray-400 text-[10px]">{p.barcode}</span>
                </td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{p.category}</td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500 truncate max-w-[120px]">{p.supplier}</td>
                <td className="px-3 py-2.5 dark:text-gray-200 text-gray-800 font-medium">€{p.price.toFixed(2)}</td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">€{p.cost.toFixed(2)}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx(
                    'font-medium',
                    p.margin >= 65 ? 'text-green-400' : p.margin >= 50 ? 'text-amber-400' : 'text-red-400',
                  )}>
                    {p.margin.toFixed(1)}%
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <span className={clsx('font-medium tabular-nums', STOCK_COLOR(p.stock))}>
                    {p.stock.toLocaleString()}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium capitalize', STATUS_STYLES[p.status])}>
                    {p.status}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="relative">
                    <button
                      onClick={() => setOpenMenu(openMenu === p.id ? null : p.id)}
                      className="opacity-0 group-hover:opacity-100 w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-all"
                    >
                      <MoreHorizontal size={14} className="dark:text-gray-400 text-gray-500" />
                    </button>
                    {openMenu === p.id && (
                      <div className="absolute right-0 top-7 z-20 w-40 dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 rounded-lg shadow-xl overflow-hidden">
                        {[
                          { icon: Edit3, label: 'Edit' },
                          { icon: Copy, label: 'Duplicate' },
                          { icon: Trash2, label: 'Delete', danger: true },
                        ].map(action => (
                          <button
                            key={action.label}
                            className={clsx(
                              'w-full flex items-center gap-2 px-3 py-2 text-xs hover:dark:bg-[#2D3748] hover:bg-gray-50 transition-colors',
                              action.danger ? 'text-red-400' : 'dark:text-gray-300 text-gray-700',
                            )}
                          >
                            <action.icon size={12} /> {action.label}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between px-5 py-2.5 border-t dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">Showing {filtered.length} of {MOCK.length} products</span>
        <div className="flex items-center gap-1">
          {[1, 2, 3, '...', 24].map((page, i) => (
            <button
              key={i}
              className={clsx(
                'w-7 h-7 rounded text-xs flex items-center justify-center transition-colors',
                page === 1
                  ? 'bg-blue-600 text-white'
                  : 'dark:text-gray-400 text-gray-500 hover:dark:bg-[#1F2937] hover:bg-gray-100',
              )}
            >
              {page}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
