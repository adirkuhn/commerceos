import { useState } from 'react'
import { ShoppingBag, TrendingUp, AlertTriangle, ChevronDown, Check, ArrowRight } from 'lucide-react'
import clsx from 'clsx'

interface ProcurementItem {
  id: string
  product: string
  sku: string
  currentStock: number
  minStock: number
  suggestedQty: number
  suppliers: {
    name: string
    cost: number
    leadDays: number
    moq: number
    recommended: boolean
  }[]
  urgency: 'critical' | 'warning' | 'planned'
  totalCost: number
}

const ITEMS: ProcurementItem[] = [
  {
    id: '1', product: 'Mouse Pad XL Gaming', sku: 'MPX-XL',
    currentStock: 8, minStock: 50, suggestedQty: 200, urgency: 'critical', totalCost: 1960,
    suppliers: [
      { name: 'GameGear BV', cost: 9.80, leadDays: 8, moq: 100, recommended: true },
      { name: 'ErgoCo NV', cost: 11.20, leadDays: 5, moq: 50, recommended: false },
    ],
  },
  {
    id: '2', product: 'LED Desk Lamp Smart', sku: 'LDL-SMT',
    currentStock: 12, minStock: 40, suggestedQty: 100, urgency: 'warning', totalCost: 2230,
    suppliers: [
      { name: 'LightTech BV', cost: 22.30, leadDays: 6, moq: 50, recommended: true },
    ],
  },
  {
    id: '3', product: 'Webcam 4K Ultra', sku: 'WCM-4K',
    currentStock: 56, minStock: 80, suggestedQty: 150, urgency: 'warning', totalCost: 14700,
    suppliers: [
      { name: 'TechWorld BV', cost: 98.00, leadDays: 7, moq: 20, recommended: true },
      { name: 'DataStore AG', cost: 103.50, leadDays: 12, moq: 10, recommended: false },
    ],
  },
  {
    id: '4', product: 'USB-C Hub 7-in-1', sku: 'UCH-007',
    currentStock: 187, minStock: 150, suggestedQty: 100, urgency: 'planned', totalCost: 3200,
    suppliers: [
      { name: 'ConnectPro Ltd', cost: 32.00, leadDays: 10, moq: 50, recommended: true },
    ],
  },
  {
    id: '5', product: 'Mechanical Keyboard TKL', sku: 'MKT-TKL',
    currentStock: 94, minStock: 100, suggestedQty: 80, urgency: 'warning', totalCost: 9920,
    suppliers: [
      { name: 'KeyMaster GmbH', cost: 124.00, leadDays: 14, moq: 20, recommended: true },
    ],
  },
]

const URGENCY_STYLE = {
  critical: 'bg-red-500/15 text-red-400',
  warning: 'bg-amber-500/15 text-amber-400',
  planned: 'bg-blue-500/15 text-blue-400',
}

const URGENCY_ICON = {
  critical: AlertTriangle,
  warning: AlertTriangle,
  planned: TrendingUp,
}

export default function Procurement() {
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [expandedSupplier, setExpandedSupplier] = useState<string | null>(null)
  const [filter, setFilter] = useState<'all' | 'critical' | 'warning' | 'planned'>('all')

  const filtered = ITEMS.filter(i => filter === 'all' || i.urgency === filter)
  const allSelected = filtered.length > 0 && filtered.every(i => selected.has(i.id))

  const toggleAll = () =>
    setSelected(allSelected ? new Set() : new Set(filtered.map(i => i.id)))

  const totalSelected = filtered
    .filter(i => selected.has(i.id))
    .reduce((s, i) => s + i.totalCost, 0)

  const totalItems = filtered.filter(i => selected.has(i.id)).length

  return (
    <div className="flex flex-col h-full">
      {/* Summary banner */}
      <div className="grid grid-cols-3 gap-0 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
        {[
          ['Critical (reorder now)', ITEMS.filter(i => i.urgency === 'critical').length.toString(), 'red'],
          ['Warning (reorder soon)', ITEMS.filter(i => i.urgency === 'warning').length.toString(), 'amber'],
          ['Planned (upcoming)', ITEMS.filter(i => i.urgency === 'planned').length.toString(), 'blue'],
        ].map(([label, count, color]) => (
          <div key={label} className="px-5 py-3 border-r last:border-0 dark:border-[#2D3748] border-gray-200">
            <p className={clsx('text-2xl font-semibold',
              color === 'red' ? 'text-red-400' : color === 'amber' ? 'text-amber-400' : 'text-blue-400',
            )}>{count}</p>
            <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">{label}</p>
          </div>
        ))}
      </div>

      {/* Toolbar */}
      <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
        <div className="flex items-center gap-1">
          {(['all', 'critical', 'warning', 'planned'] as const).map(f => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={clsx(
                'px-3 py-1.5 rounded-lg text-xs font-medium capitalize transition-colors',
                filter === f
                  ? 'dark:bg-[#1F2937] bg-gray-100 dark:text-gray-100 text-gray-800'
                  : 'dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600',
              )}
            >
              {f}
            </button>
          ))}
        </div>

        {selected.size > 0 && (
          <div className="flex items-center gap-3 ml-auto">
            <span className="text-xs dark:text-gray-400 text-gray-500">
              {totalItems} items · <span className="font-medium dark:text-gray-200 text-gray-800">€{totalSelected.toLocaleString()}</span> total
            </span>
            <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700">
              <ShoppingBag size={13} /> Generate Purchase Orders
            </button>
          </div>
        )}

        {selected.size === 0 && (
          <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 ml-auto">
            <ShoppingBag size={13} /> Order All Critical
          </button>
        )}
      </div>

      {/* Items */}
      <div className="flex-1 overflow-auto p-4 space-y-3">
        {/* Select all */}
        <div className="flex items-center gap-2 px-1">
          <input
            type="checkbox"
            checked={allSelected}
            onChange={toggleAll}
            className="rounded accent-blue-600 w-3 h-3"
          />
          <span className="text-xs dark:text-gray-500 text-gray-400">Select all {filtered.length} items</span>
        </div>

        {filtered.map(item => {
          const UrgIcon = URGENCY_ICON[item.urgency]
          const stockPct = Math.min((item.currentStock / item.minStock) * 100, 100)
          const isExpanded = expandedSupplier === item.id

          return (
            <div
              key={item.id}
              className={clsx(
                'dark:bg-[#1F2937] bg-white rounded-xl border overflow-hidden transition-all',
                selected.has(item.id)
                  ? 'dark:border-blue-500/30 border-blue-300'
                  : 'dark:border-[#2D3748] border-gray-200',
              )}
            >
              {/* Main row */}
              <div className="flex items-center gap-4 p-4">
                <input
                  type="checkbox"
                  checked={selected.has(item.id)}
                  onChange={() => setSelected(prev => {
                    const s = new Set(prev)
                    s.has(item.id) ? s.delete(item.id) : s.add(item.id)
                    return s
                  })}
                  className="rounded accent-blue-600 w-3 h-3 shrink-0"
                />

                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <p className="text-sm font-medium dark:text-gray-100 text-gray-900">{item.product}</p>
                    <span className="font-mono text-[10px] dark:text-gray-500 text-gray-400">{item.sku}</span>
                    <span className={clsx('flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium', URGENCY_STYLE[item.urgency])}>
                      <UrgIcon size={9} />
                      {item.urgency}
                    </span>
                  </div>

                  {/* Stock bar */}
                  <div className="flex items-center gap-2">
                    <div className="flex-1 h-1.5 dark:bg-[#2D3748] bg-gray-200 rounded-full overflow-hidden">
                      <div
                        className={clsx('h-full rounded-full transition-all',
                          stockPct < 20 ? 'bg-red-500' : stockPct < 60 ? 'bg-amber-500' : 'bg-green-500',
                        )}
                        style={{ width: `${stockPct}%` }}
                      />
                    </div>
                    <span className="text-[10px] dark:text-gray-400 text-gray-500 shrink-0">
                      {item.currentStock} / {item.minStock} min
                    </span>
                  </div>
                </div>

                {/* Suggested qty & cost */}
                <div className="text-right shrink-0 mr-4">
                  <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">{item.suggestedQty} units</p>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400">suggested order</p>
                </div>
                <div className="text-right shrink-0 mr-2">
                  <p className="text-sm font-semibold dark:text-gray-200 text-gray-800">€{item.totalCost.toLocaleString()}</p>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400">total cost</p>
                </div>

                {/* Supplier toggle */}
                <button
                  onClick={() => setExpandedSupplier(isExpanded ? null : item.id)}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg dark:bg-[#2D3748] bg-gray-100 text-xs dark:text-gray-300 text-gray-600 hover:dark:bg-[#374151] hover:bg-gray-200 transition-colors shrink-0"
                >
                  {item.suppliers.length} supplier{item.suppliers.length > 1 ? 's' : ''}
                  <ChevronDown size={12} className={clsx('transition-transform', isExpanded && 'rotate-180')} />
                </button>
              </div>

              {/* Supplier comparison */}
              {isExpanded && (
                <div className="border-t dark:border-[#2D3748] border-gray-100">
                  <table className="w-full text-xs">
                    <thead>
                      <tr className="dark:bg-[#111827]/50 bg-gray-50">
                        <th className="px-4 py-2 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">Supplier</th>
                        <th className="px-4 py-2 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">Unit Cost</th>
                        <th className="px-4 py-2 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">Lead Time</th>
                        <th className="px-4 py-2 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">MOQ</th>
                        <th className="px-4 py-2 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">Total</th>
                        <th className="px-4 py-2 w-32" />
                      </tr>
                    </thead>
                    <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
                      {item.suppliers.map(sup => (
                        <tr key={sup.name} className={clsx(
                          'transition-colors',
                          sup.recommended ? 'dark:bg-green-500/5 bg-green-50' : 'hover:dark:bg-[#2D3748]/30 hover:bg-gray-50',
                        )}>
                          <td className="px-4 py-2.5">
                            <div className="flex items-center gap-1.5">
                              {sup.recommended && <Check size={11} className="text-green-400" />}
                              <span className={clsx('font-medium', sup.recommended ? 'text-green-400' : 'dark:text-gray-300 text-gray-700')}>
                                {sup.name}
                              </span>
                              {sup.recommended && <span className="text-[9px] bg-green-500/20 text-green-400 px-1.5 py-0.5 rounded-full">recommended</span>}
                            </div>
                          </td>
                          <td className="px-4 py-2.5 dark:text-gray-200 text-gray-800 font-medium">€{sup.cost.toFixed(2)}</td>
                          <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">{sup.leadDays}d</td>
                          <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">{sup.moq} units</td>
                          <td className="px-4 py-2.5 dark:text-gray-200 text-gray-800 font-medium">
                            €{(sup.cost * item.suggestedQty).toLocaleString()}
                          </td>
                          <td className="px-4 py-2.5">
                            <button className="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-blue-600 text-white text-[10px] font-medium hover:bg-blue-700">
                              Order <ArrowRight size={10} />
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
