import { useState } from 'react'
import {
  Search, Plus, Upload, MoreHorizontal, Globe, Package,
  TrendingDown, Star, ExternalLink,
} from 'lucide-react'
import clsx from 'clsx'

interface Supplier {
  id: string
  name: string
  country: string
  flag: string
  products: number
  lastOrder: string
  avgCost: number
  leadDays: number
  rating: number
  status: 'active' | 'inactive' | 'pending'
  categories: string[]
}

const SUPPLIERS: Supplier[] = [
  { id: '1', name: 'TechWorld BV', country: 'Netherlands', flag: '🇳🇱', products: 84, lastOrder: '2026-05-12', avgCost: 94.50, leadDays: 7, rating: 4.8, status: 'active', categories: ['Electronics', 'Accessories'] },
  { id: '2', name: 'ConnectPro Ltd', country: 'United Kingdom', flag: '🇬🇧', products: 47, lastOrder: '2026-05-08', avgCost: 32.20, leadDays: 10, rating: 4.5, status: 'active', categories: ['Cables', 'Accessories'] },
  { id: '3', name: 'KeyMaster GmbH', country: 'Germany', flag: '🇩🇪', products: 23, lastOrder: '2026-04-30', avgCost: 121.00, leadDays: 14, rating: 4.2, status: 'active', categories: ['Peripherals'] },
  { id: '4', name: 'ErgoCo NV', country: 'Belgium', flag: '🇧🇪', products: 31, lastOrder: '2026-05-01', avgCost: 28.40, leadDays: 5, rating: 4.7, status: 'active', categories: ['Accessories', 'Furniture'] },
  { id: '5', name: 'DataStore AG', country: 'Switzerland', flag: '🇨🇭', products: 18, lastOrder: '2026-04-15', avgCost: 68.90, leadDays: 12, rating: 3.9, status: 'active', categories: ['Storage'] },
  { id: '6', name: 'GameGear BV', country: 'Netherlands', flag: '🇳🇱', products: 12, lastOrder: '2026-03-20', avgCost: 9.80, leadDays: 8, rating: 4.1, status: 'inactive', categories: ['Accessories', 'Gaming'] },
  { id: '7', name: 'LightTech BV', country: 'Netherlands', flag: '🇳🇱', products: 29, lastOrder: '2026-05-10', avgCost: 22.30, leadDays: 6, rating: 4.6, status: 'active', categories: ['Lighting', 'Smart Home'] },
  { id: '8', name: 'AudioPro SRL', country: 'Italy', flag: '🇮🇹', products: 0, lastOrder: '—', avgCost: 0, leadDays: 21, rating: 0, status: 'pending', categories: ['Audio'] },
]

const STATUS_STYLE = {
  active: 'bg-green-500/15 text-green-400',
  inactive: 'bg-gray-500/15 text-gray-400',
  pending: 'bg-amber-500/15 text-amber-400',
}

const PRICE_HISTORY = [
  { month: 'Dec', cost: 97.20 },
  { month: 'Jan', cost: 95.80 },
  { month: 'Feb', cost: 96.40 },
  { month: 'Mar', cost: 93.10 },
  { month: 'Apr', cost: 94.80 },
  { month: 'May', cost: 94.50 },
]

const SUP_PRODUCTS = [
  { sku: 'WHP-001', name: 'Wireless Headphones Pro', cost: 89.50, price: 249.99, stock: 423 },
  { sku: 'WCM-4K', name: 'Webcam 4K Ultra', cost: 98.00, price: 299.99, stock: 56 },
]

function Stars({ rating }: { rating: number }) {
  return (
    <div className="flex items-center gap-0.5">
      {[1, 2, 3, 4, 5].map(s => (
        <Star
          key={s}
          size={10}
          className={s <= Math.round(rating) ? 'fill-amber-400 text-amber-400' : 'dark:text-gray-600 text-gray-300'}
        />
      ))}
      <span className="ml-1 text-[10px] dark:text-gray-400 text-gray-500">{rating > 0 ? rating.toFixed(1) : '—'}</span>
    </div>
  )
}

export default function Suppliers() {
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<Supplier | null>(null)

  const filtered = SUPPLIERS.filter(s =>
    s.name.toLowerCase().includes(search.toLowerCase()) ||
    s.country.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="flex h-full overflow-hidden">
      {/* Main list */}
      <div className={clsx('flex flex-col', selected ? 'flex-1' : 'w-full')}>
        {/* Toolbar */}
        <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
          <div className="relative flex-1 max-w-sm">
            <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
            <input
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search suppliers..."
              className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
          <div className="flex items-center gap-2 ml-auto">
            <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
              <Upload size={13} /> Import Catalog
            </button>
            <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 font-medium">
              <Plus size={13} /> Add Supplier
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
                {['Supplier', 'Country', 'Categories', 'Products', 'Avg Cost', 'Lead Time', 'Rating', 'Last Order', 'Status', ''].map(h => (
                  <th key={h} className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider whitespace-nowrap">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
              {filtered.map(sup => (
                <tr
                  key={sup.id}
                  onClick={() => setSelected(selected?.id === sup.id ? null : sup)}
                  className={clsx(
                    'cursor-pointer hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors group',
                    selected?.id === sup.id && 'dark:bg-blue-600/5 bg-blue-50',
                  )}
                >
                  <td className="px-4 py-2.5" onClick={e => e.stopPropagation()}>
                    <input type="checkbox" className="rounded accent-blue-600 w-3 h-3" />
                  </td>
                  <td className="px-3 py-2.5">
                    <div className="flex items-center gap-2">
                      <div className="w-7 h-7 rounded-lg dark:bg-[#2D3748] bg-gray-100 flex items-center justify-center shrink-0 text-sm">
                        {sup.flag}
                      </div>
                      <span className="font-medium dark:text-gray-100 text-gray-900">{sup.name}</span>
                    </div>
                  </td>
                  <td className="px-3 py-2.5">
                    <div className="flex items-center gap-1">
                      <Globe size={11} className="dark:text-gray-500 text-gray-400" />
                      <span className="dark:text-gray-400 text-gray-500">{sup.country}</span>
                    </div>
                  </td>
                  <td className="px-3 py-2.5">
                    <div className="flex flex-wrap gap-1">
                      {sup.categories.slice(0, 2).map(c => (
                        <span key={c} className="px-1.5 py-0.5 rounded text-[9px] dark:bg-[#2D3748] bg-gray-100 dark:text-gray-400 text-gray-500">{c}</span>
                      ))}
                    </div>
                  </td>
                  <td className="px-3 py-2.5">
                    <div className="flex items-center gap-1">
                      <Package size={11} className="dark:text-gray-500 text-gray-400" />
                      <span className="dark:text-gray-300 text-gray-700">{sup.products}</span>
                    </div>
                  </td>
                  <td className="px-3 py-2.5 dark:text-gray-200 text-gray-800 font-medium">
                    {sup.avgCost > 0 ? `€${sup.avgCost.toFixed(2)}` : '—'}
                  </td>
                  <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{sup.leadDays}d</td>
                  <td className="px-3 py-2.5"><Stars rating={sup.rating} /></td>
                  <td className="px-3 py-2.5 dark:text-gray-500 text-gray-400 font-mono text-[10px]">{sup.lastOrder}</td>
                  <td className="px-3 py-2.5">
                    <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium capitalize', STATUS_STYLE[sup.status])}>
                      {sup.status}
                    </span>
                  </td>
                  <td className="px-3 py-2.5">
                    <button
                      onClick={e => e.stopPropagation()}
                      className="opacity-0 group-hover:opacity-100 w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200"
                    >
                      <MoreHorizontal size={14} className="dark:text-gray-400 text-gray-500" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Supplier detail panel */}
      {selected && (
        <div className="w-96 border-l dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 flex flex-col overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <div className="flex items-center gap-2">
              <span className="text-lg">{selected.flag}</span>
              <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">{selected.name}</h2>
            </div>
            <button onClick={() => setSelected(null)} className="text-xs dark:text-gray-500 text-gray-400">✕</button>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-3 gap-0 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
            {[
              ['Products', selected.products.toString()],
              ['Lead Time', `${selected.leadDays}d`],
              ['Avg Cost', selected.avgCost > 0 ? `€${selected.avgCost.toFixed(0)}` : '—'],
            ].map(([label, val]) => (
              <div key={label} className="px-4 py-3 border-r last:border-0 dark:border-[#2D3748] border-gray-200 text-center">
                <p className="text-base font-semibold dark:text-gray-100 text-gray-900">{val}</p>
                <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">{label}</p>
              </div>
            ))}
          </div>

          {/* Price history */}
          <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
            <p className="text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-2">Cost History (avg)</p>
            <div className="flex items-end gap-1 h-14">
              {PRICE_HISTORY.map((d, i) => {
                const min = Math.min(...PRICE_HISTORY.map(x => x.cost))
                const max = Math.max(...PRICE_HISTORY.map(x => x.cost))
                const pct = ((d.cost - min) / (max - min)) * 60 + 40
                return (
                  <div key={i} className="flex-1 flex flex-col items-center gap-1">
                    <div
                      className="w-full rounded-sm bg-blue-500/60"
                      style={{ height: `${pct}%` }}
                    />
                    <span className="text-[9px] dark:text-gray-600 text-gray-400">{d.month}</span>
                  </div>
                )
              })}
            </div>
            <div className="flex items-center justify-between mt-1">
              <span className="text-[10px] dark:text-gray-500 text-gray-400">€{Math.min(...PRICE_HISTORY.map(x => x.cost)).toFixed(2)} min</span>
              <div className="flex items-center gap-1 text-[10px] text-green-400">
                <TrendingDown size={10} /> -2.9% vs Dec
              </div>
            </div>
          </div>

          {/* Products */}
          <div className="flex-1 overflow-y-auto">
            <p className="px-4 py-2 text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider border-b dark:border-[#2D3748] border-gray-200">
              Products ({selected.products})
            </p>
            {SUP_PRODUCTS.map(p => (
              <div key={p.sku} className="flex items-center gap-3 px-4 py-2.5 border-b dark:border-[#2D3748]/50 border-gray-100">
                <div className="flex-1 min-w-0">
                  <p className="text-xs font-medium dark:text-gray-100 text-gray-900 truncate">{p.name}</p>
                  <p className="text-[10px] font-mono dark:text-gray-500 text-gray-400">{p.sku}</p>
                </div>
                <div className="text-right shrink-0">
                  <p className="text-xs font-medium dark:text-gray-200 text-gray-800">€{p.cost.toFixed(2)}</p>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400">cost</p>
                </div>
              </div>
            ))}
            {selected.products === 0 && (
              <p className="px-4 py-6 text-xs dark:text-gray-600 text-gray-400 text-center">No products yet</p>
            )}
          </div>

          {/* Actions */}
          <div className="p-4 border-t dark:border-[#2D3748] border-gray-200 space-y-2 shrink-0">
            <button className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors">
              Create Purchase Order
            </button>
            <button className="w-full py-2 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 text-xs flex items-center justify-center gap-2">
              <ExternalLink size={12} /> View Full Profile
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
