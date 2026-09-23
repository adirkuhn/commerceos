import { Truck, Package, Search, CheckCircle, Clock, AlertTriangle, Printer, ExternalLink } from 'lucide-react'
import clsx from 'clsx'
import { useLocation, useNavigate } from 'react-router-dom'
import { subPathSegments } from '../routes'
import { useQueryString } from '../routing/useQueryState'

interface Shipment {
  id: string
  orderId: string
  customer: string
  address: string
  carrier: 'sendcloud' | 'myparcel' | 'packlink' | 'dhl' | 'postnl'
  service: string
  weight: string
  status: 'label_created' | 'picked_up' | 'in_transit' | 'out_for_delivery' | 'delivered' | 'exception'
  tracking: string
  created: string
  cost: number
}

const SHIPMENTS: Shipment[] = [
  { id: 'SHP-0841', orderId: 'ORD-10039', customer: 'Marco Rossi', address: 'Milan, IT', carrier: 'dhl', service: 'DHL Express', weight: '0.8kg', status: 'in_transit', tracking: '1Z999AA10123456784', created: '2026-05-16', cost: 12.50 },
  { id: 'SHP-0840', orderId: 'ORD-10038', customer: 'Emma Wilson', address: 'London, GB', carrier: 'sendcloud', service: 'Sendcloud Priority', weight: '1.2kg', status: 'delivered', tracking: 'SC9482710293', created: '2026-05-15', cost: 9.80 },
  { id: 'SHP-0839', orderId: 'ORD-10037', customer: 'Pieter Vos', address: 'Utrecht, NL', carrier: 'postnl', service: 'PostNL Standard', weight: '0.3kg', status: 'delivered', tracking: '3STOLK9002100', created: '2026-05-14', cost: 4.95 },
  { id: 'SHP-0838', orderId: 'ORD-10034', customer: 'Mia Schmidt', address: 'Hamburg, DE', carrier: 'myparcel', service: 'MyParcel Standard', weight: '0.6kg', status: 'picked_up', tracking: '6S90000001', created: '2026-05-15', cost: 6.75 },
  { id: 'SHP-0837', orderId: 'ORD-10033', customer: 'Kevin Smit', address: 'Rotterdam, NL', carrier: 'postnl', service: 'PostNL Standard', weight: '0.5kg', status: 'label_created', tracking: '3STOLK9002099', created: '2026-05-16', cost: 4.95 },
  { id: 'SHP-0836', orderId: 'ORD-10031', customer: 'Claire Dubois', address: 'Lyon, FR', carrier: 'packlink', service: 'Packlink Standard', weight: '2.1kg', status: 'exception', tracking: 'PL209481029', created: '2026-05-13', cost: 14.20 },
]

const CARRIER_COLORS: Record<string, string> = {
  sendcloud: 'bg-purple-500/15 text-purple-400',
  myparcel: 'bg-blue-500/15 text-blue-400',
  packlink: 'bg-orange-500/15 text-orange-400',
  dhl: 'bg-yellow-500/15 text-yellow-500',
  postnl: 'bg-orange-600/15 text-orange-500',
}

const STATUS_CONFIG: Record<string, { label: string; color: string; icon: React.ElementType }> = {
  label_created: { label: 'Label Created', color: 'text-gray-400', icon: Printer },
  picked_up: { label: 'Picked Up', color: 'text-blue-400', icon: Package },
  in_transit: { label: 'In Transit', color: 'text-amber-400', icon: Truck },
  out_for_delivery: { label: 'Out for Delivery', color: 'text-blue-400', icon: Truck },
  delivered: { label: 'Delivered', color: 'text-green-400', icon: CheckCircle },
  exception: { label: 'Exception', color: 'text-red-400', icon: AlertTriangle },
}

const CARRIER_RULES = [
  { name: 'PostNL Standard', carrier: 'postnl', condition: 'NL · < 1kg', cost: '€4.95', auto: true },
  { name: 'PostNL Priority', carrier: 'postnl', condition: 'NL · 1–10kg', cost: '€8.50', auto: true },
  { name: 'DHL Express', carrier: 'dhl', condition: 'EU · express', cost: '€12.50', auto: false },
  { name: 'Sendcloud Priority', carrier: 'sendcloud', condition: 'EU · standard', cost: '€9.80', auto: true },
  { name: 'MyParcel Standard', carrier: 'myparcel', condition: 'DE · < 5kg', cost: '€6.75', auto: true },
]

const SHIPPING_PATH = '/shipping'

export default function Shipping() {
  // Tab is the path (/shipping, /shipping/rules); search is ?q= -- refresh
  // and Back keep them.
  const [searchParam, setSearch] = useQueryString('q')
  const search = searchParam ?? ''
  const location = useLocation()
  const navigate = useNavigate()
  const tab: 'shipments' | 'rules' = subPathSegments(location.pathname, SHIPPING_PATH)[0] === 'rules' ? 'rules' : 'shipments'
  const setTab = (next: 'shipments' | 'rules') => {
    if (next !== tab) navigate(next === 'rules' ? `${SHIPPING_PATH}/rules` : SHIPPING_PATH)
  }

  const filtered = SHIPMENTS.filter(s =>
    s.orderId.includes(search) || s.customer.toLowerCase().includes(search.toLowerCase()) || s.tracking.includes(search)
  )

  const stats = {
    today: SHIPMENTS.filter(s => s.created === '2026-05-16').length,
    inTransit: SHIPMENTS.filter(s => s.status === 'in_transit' || s.status === 'out_for_delivery' || s.status === 'picked_up').length,
    delivered: SHIPMENTS.filter(s => s.status === 'delivered').length,
    exceptions: SHIPMENTS.filter(s => s.status === 'exception').length,
  }

  return (
    <div className="flex flex-col h-full">
      {/* Stats */}
      <div className="grid grid-cols-4 gap-0 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
        {[
          ['Shipped Today', stats.today.toString(), 'blue'],
          ['In Transit', stats.inTransit.toString(), 'amber'],
          ['Delivered', stats.delivered.toString(), 'green'],
          ['Exceptions', stats.exceptions.toString(), 'red'],
        ].map(([label, val, color]) => (
          <div key={label} className="px-5 py-3 border-r last:border-0 dark:border-[#2D3748] border-gray-200">
            <p className={clsx('text-2xl font-semibold',
              color === 'blue' ? 'text-blue-400' : color === 'amber' ? 'text-amber-400' : color === 'green' ? 'text-green-400' : 'text-red-400',
            )}>{val}</p>
            <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">{label}</p>
          </div>
        ))}
      </div>

      {/* Tabs + toolbar */}
      <div className="flex items-center gap-4 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
        {(['shipments', 'rules'] as const).map(t => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={clsx(
              'text-xs font-medium capitalize transition-colors pb-0.5',
              tab === t
                ? 'dark:text-gray-100 text-gray-900 border-b-2 border-blue-500'
                : 'dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600',
            )}
          >
            {t === 'shipments' ? 'Shipments' : 'Carrier Rules'}
          </button>
        ))}

        {tab === 'shipments' && (
          <div className="relative ml-4">
            <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
            <input
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Order, customer, tracking..."
              className="pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500 w-60"
            />
          </div>
        )}
      </div>

      {/* Shipments table */}
      {tab === 'shipments' && (
        <div className="flex-1 overflow-auto">
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
              <tr>
                <th className="px-4 py-2.5 w-8">
                  <input type="checkbox" className="rounded accent-blue-600 w-3 h-3" />
                </th>
                {['Shipment', 'Order', 'Customer', 'Carrier', 'Service', 'Weight', 'Tracking', 'Status', 'Cost', 'Date', ''].map(h => (
                  <th key={h} className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider whitespace-nowrap">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
              {filtered.map(s => {
                const cfg = STATUS_CONFIG[s.status]
                const Icon = cfg.icon
                return (
                  <tr key={s.id} className="hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors group">
                    <td className="px-4 py-2.5">
                      <input type="checkbox" className="rounded accent-blue-600 w-3 h-3" />
                    </td>
                    <td className="px-3 py-2.5 font-mono font-medium dark:text-gray-200 text-gray-800">{s.id}</td>
                    <td className="px-3 py-2.5 font-mono dark:text-gray-400 text-gray-500 text-[10px]">{s.orderId}</td>
                    <td className="px-3 py-2.5 dark:text-gray-300 text-gray-700">{s.customer}</td>
                    <td className="px-3 py-2.5">
                      <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium capitalize', CARRIER_COLORS[s.carrier])}>
                        {s.carrier}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{s.service}</td>
                    <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{s.weight}</td>
                    <td className="px-3 py-2.5">
                      <span className="font-mono text-[10px] dark:text-gray-500 text-gray-400">{s.tracking}</span>
                    </td>
                    <td className="px-3 py-2.5">
                      <span className={clsx('flex items-center gap-1 w-fit font-medium text-[10px]', cfg.color)}>
                        <Icon size={10} />
                        {cfg.label}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 dark:text-gray-200 text-gray-800 font-medium">€{s.cost.toFixed(2)}</td>
                    <td className="px-3 py-2.5 dark:text-gray-500 text-gray-400 font-mono text-[10px]">{s.created}</td>
                    <td className="px-3 py-2.5">
                      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100">
                        <button className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200">
                          <Printer size={12} className="dark:text-gray-400 text-gray-500" />
                        </button>
                        <button className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200">
                          <ExternalLink size={12} className="dark:text-gray-400 text-gray-500" />
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Carrier rules */}
      {tab === 'rules' && (
        <div className="flex-1 overflow-auto p-5">
          <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
            <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
              <div>
                <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Auto-Carrier Rules</h3>
                <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">Rules applied automatically based on weight, region, and cost</p>
              </div>
              <button className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700">
                + Add Rule
              </button>
            </div>
            <table className="w-full text-xs">
              <thead className="dark:bg-[#111827]/50 bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
                <tr>
                  {['Service', 'Carrier', 'Condition', 'Cost', 'Auto', 'Priority'].map(h => (
                    <th key={h} className="px-4 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
                {CARRIER_RULES.map((rule, i) => (
                  <tr key={rule.name} className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50 transition-colors">
                    <td className="px-4 py-3 dark:text-gray-200 text-gray-800 font-medium">{rule.name}</td>
                    <td className="px-4 py-3">
                      <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium', CARRIER_COLORS[rule.carrier])}>
                        {rule.carrier}
                      </span>
                    </td>
                    <td className="px-4 py-3 dark:text-gray-400 text-gray-500">{rule.condition}</td>
                    <td className="px-4 py-3 dark:text-gray-200 text-gray-800 font-medium">{rule.cost}</td>
                    <td className="px-4 py-3">
                      {rule.auto
                        ? <CheckCircle size={14} className="text-green-400" />
                        : <Clock size={14} className="dark:text-gray-500 text-gray-400" />
                      }
                    </td>
                    <td className="px-4 py-3">
                      <span className="dark:text-gray-400 text-gray-500 text-[10px]">#{i + 1}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
