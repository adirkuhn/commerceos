import { Truck, Package, Search, CheckCircle, Clock, AlertTriangle, Printer, ExternalLink } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
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
  myparcel: 'bg-accent/15 text-accent',
  packlink: 'bg-orange-500/15 text-orange-400',
  dhl: 'bg-yellow-500/15 text-warning',
  postnl: 'bg-orange-600/15 text-orange-500',
}

const STATUS_CONFIG: Record<Shipment['status'], { color: string; icon: React.ElementType }> = {
  label_created: { color: 'text-ink-4', icon: Printer },
  picked_up: { color: 'text-accent', icon: Package },
  in_transit: { color: 'text-warning', icon: Truck },
  out_for_delivery: { color: 'text-accent', icon: Truck },
  delivered: { color: 'text-success', icon: CheckCircle },
  exception: { color: 'text-danger', icon: AlertTriangle },
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
  const { t } = useTranslation(['shipping', 'common'])
  const statusLabel: Record<Shipment['status'], string> = {
    label_created: t('status.label_created'), picked_up: t('status.picked_up'), in_transit: t('status.in_transit'),
    out_for_delivery: t('status.out_for_delivery'), delivered: t('status.delivered'), exception: t('status.exception'),
  }
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
      <div className="grid grid-cols-4 gap-0 border-b border-line shrink-0">
        {[
          [t('page.shippedToday'), stats.today.toString(), 'blue'],
          [t('page.inTransit'), stats.inTransit.toString(), 'amber'],
          [t('page.delivered'), stats.delivered.toString(), 'green'],
          [t('page.exceptions'), stats.exceptions.toString(), 'red'],
        ].map(([label, val, color]) => (
          <div key={label} className="px-5 py-3 border-r last:border-0 border-line">
            <p className={clsx('text-2xl font-semibold',
              color === 'blue' ? 'text-accent' : color === 'amber' ? 'text-warning' : color === 'green' ? 'text-success' : 'text-danger',
            )}>{val}</p>
            <p className="text-2xs text-ink-4 mt-0.5">{label}</p>
          </div>
        ))}
      </div>

      {/* Tabs + toolbar */}
      <div className="flex items-center gap-4 px-5 py-2.5 border-b border-line shrink-0">
        {(['shipments', 'rules'] as const).map(item => (
          <button aria-pressed={tab === item}
            key={item}
            onClick={() => setTab(item)}
            className={clsx(
              'text-xs font-medium capitalize transition-colors pb-0.5',
              tab === item
                ? 'text-ink border-b-2 border-accent'
                : 'text-ink-4 hover:text-ink-2',
            )}
          >
            {item === 'shipments' ? t('page.tabShipments') : t('page.tabRules')}
          </button>
        ))}

        {tab === 'shipments' && (
          <div className="relative ml-4">
            <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
            <input
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder={t('page.search')}
              className="pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus w-60"
            />
          </div>
        )}
      </div>

      {/* Shipments table */}
      {tab === 'shipments' && (
        <div className="flex-1 overflow-auto">
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
              <tr>
                <th className="px-4 py-2.5 w-8">
                  <input type="checkbox" className="rounded accent-accent-solid w-3 h-3" />
                </th>
                {[t('page.colShipment'), t('page.colOrder'), t('page.colCustomer'), t('page.colCarrier'), t('page.colService'), t('page.colWeight'), t('page.colTracking'), t('page.colStatus'), t('page.colCost'), t('page.colDate'), ''].map((h, i) => (
                  <th key={i} className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider whitespace-nowrap">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-line/60">
              {filtered.map(s => {
                const cfg = STATUS_CONFIG[s.status]
                const Icon = cfg.icon
                return (
                  <tr key={s.id} className="hover:bg-hover/60 transition-colors group">
                    <td className="px-4 py-2.5">
                      <input type="checkbox" className="rounded accent-accent-solid w-3 h-3" />
                    </td>
                    <td className="px-3 py-2.5 font-mono font-medium text-ink">{s.id}</td>
                    <td className="px-3 py-2.5 font-mono text-ink-3 text-2xs">{s.orderId}</td>
                    <td className="px-3 py-2.5 text-ink-2">{s.customer}</td>
                    <td className="px-3 py-2.5">
                      <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium capitalize', CARRIER_COLORS[s.carrier])}>
                        {s.carrier}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-ink-3">{s.service}</td>
                    <td className="px-3 py-2.5 text-ink-3">{s.weight}</td>
                    <td className="px-3 py-2.5">
                      <span className="font-mono text-2xs text-ink-4">{s.tracking}</span>
                    </td>
                    <td className="px-3 py-2.5">
                      <span className={clsx('flex items-center gap-1 w-fit font-medium text-2xs', cfg.color)}>
                        <Icon size={10} />
                        {statusLabel[s.status]}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-ink font-medium">€{s.cost.toFixed(2)}</td>
                    <td className="px-3 py-2.5 text-ink-4 font-mono text-2xs">{s.created}</td>
                    <td className="px-3 py-2.5">
                      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100">
                        <button aria-label={t('page.printLabel')} title={t('page.printLabel')} className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover">
                          <Printer size={12} className="text-ink-3" />
                        </button>
                        <button aria-label={t('page.openTracking')} title={t('page.openTracking')} className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover">
                          <ExternalLink size={12} className="text-ink-3" />
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
          <div className="bg-raised rounded-xl border border-line overflow-hidden">
            <div className="flex items-center justify-between px-4 py-3 border-b border-line">
              <div>
                <h3 className="text-sm font-semibold text-ink">{t('page.rulesTitle')}</h3>
                <p className="text-2xs text-ink-4 mt-0.5">{t('page.rulesIntro')}</p>
              </div>
              <button className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover">
                {t('page.addRule')}
              </button>
            </div>
            <table className="w-full text-xs">
              <thead className="dark:bg-night-900/50 bg-gray-50 border-b border-line">
                <tr>
                  {[t('page.colService'), t('page.colCarrier'), t('page.colCondition'), t('page.colCost'), t('page.colAuto'), t('page.colPriority')].map((h, i) => (
                    <th key={i} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-line/50">
                {CARRIER_RULES.map((rule, i) => (
                  <tr key={rule.name} className="hover:bg-hover/30 transition-colors">
                    <td className="px-4 py-3 text-ink font-medium">{rule.name}</td>
                    <td className="px-4 py-3">
                      <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium', CARRIER_COLORS[rule.carrier])}>
                        {rule.carrier}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-ink-3">{rule.condition}</td>
                    <td className="px-4 py-3 text-ink font-medium">{rule.cost}</td>
                    <td className="px-4 py-3">
                      {rule.auto
                        ? <CheckCircle size={14} className="text-success" />
                        : <Clock size={14} className="text-ink-4" />
                      }
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-ink-3 text-2xs">#{i + 1}</span>
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
