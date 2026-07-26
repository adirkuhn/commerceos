import { useState } from 'react'
import { CheckCircle, XCircle, AlertTriangle, RefreshCw, Settings, ExternalLink, Plug } from 'lucide-react'
import clsx from 'clsx'

interface Integration {
  id: string
  name: string
  type: 'marketplace' | 'store' | 'shipping'
  status: 'connected' | 'error' | 'paused'
  lastSync: string
  syncedOrders: number
  syncedProducts: number
  error?: string
}

const INTEGRATIONS: Integration[] = [
  { id: 'shopify', name: 'Shopify', type: 'store', status: 'connected', lastSync: '2m ago', syncedOrders: 8420, syncedProducts: 234 },
  { id: 'woocommerce', name: 'WooCommerce', type: 'store', status: 'connected', lastSync: '5m ago', syncedOrders: 3120, syncedProducts: 189 },
  { id: 'bol', name: 'Bol.com', type: 'marketplace', status: 'error', lastSync: '1h ago', syncedOrders: 1240, syncedProducts: 87, error: 'API rate limit exceeded (429). Retry in 45min.' },
  { id: 'lightspeed', name: 'Lightspeed', type: 'store', status: 'connected', lastSync: '12m ago', syncedOrders: 987, syncedProducts: 145 },
]

const CHANNEL_COLORS: Record<string, string> = {
  shopify: 'from-green-500 to-emerald-600',
  woocommerce: 'from-purple-500 to-violet-600',
  bol: 'from-blue-500 to-blue-600',
  lightspeed: 'from-orange-500 to-amber-500',
}

const STATUS_ICON = {
  connected: CheckCircle,
  error: XCircle,
  paused: AlertTriangle,
}

const STATUS_COLOR = {
  connected: 'text-green-400',
  error: 'text-red-400',
  paused: 'text-amber-400',
}

const MOCK_LOGS = [
  { time: '14:32:01', integration: 'Shopify', type: 'sync', msg: 'Pulled 8 new orders', ok: true },
  { time: '14:32:01', integration: 'Shopify', type: 'sync', msg: 'Updated stock for 3 SKUs', ok: true },
  { time: '14:18:44', integration: 'WooCommerce', type: 'sync', msg: 'Pulled 2 new orders', ok: true },
  { time: '13:01:22', integration: 'Bol.com', type: 'error', msg: 'Rate limit 429 — backing off', ok: false },
  { time: '13:01:21', integration: 'Bol.com', type: 'retry', msg: 'Retry #3 failed', ok: false },
  { time: '12:55:10', integration: 'Lightspeed', type: 'sync', msg: 'Pulled 1 new order', ok: true },
  { time: '12:50:33', integration: 'Shopify', type: 'webhook', msg: 'Order cancelled webhook received', ok: true },
]

export default function Integrations() {
  const [selectedIntegration, setSelectedIntegration] = useState<string | null>(null)
  const selected = INTEGRATIONS.find(i => i.id === selectedIntegration)

  return (
    <div className="flex h-full overflow-hidden">
      {/* Left: integration cards + logs */}
      <div className="flex-1 flex flex-col overflow-hidden p-5 gap-5">
        {/* Cards */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
          {INTEGRATIONS.map(intg => {
            const Icon = STATUS_ICON[intg.status]
            return (
              <button
                key={intg.id}
                onClick={() => setSelectedIntegration(selectedIntegration === intg.id ? null : intg.id)}
                className={clsx(
                  'text-left p-4 rounded-xl border transition-all',
                  selectedIntegration === intg.id
                    ? 'dark:border-blue-500/50 border-blue-400 dark:bg-blue-500/5 bg-blue-50'
                    : 'dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white hover:dark:border-[#374151] hover:border-gray-300',
                  intg.status === 'error' && selectedIntegration !== intg.id && 'dark:border-red-500/30 border-red-200',
                )}
              >
                <div className="flex items-start justify-between mb-3">
                  <div className={clsx(
                    'w-9 h-9 rounded-lg bg-gradient-to-br flex items-center justify-center text-white font-bold text-sm',
                    CHANNEL_COLORS[intg.id],
                  )}>
                    {intg.name[0]}
                  </div>
                  <Icon size={14} className={STATUS_COLOR[intg.status]} />
                </div>
                <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">{intg.name}</p>
                <p className="text-[10px] dark:text-gray-500 text-gray-400 capitalize mt-0.5">{intg.type}</p>
                {intg.status === 'error' && (
                  <div className="mt-2 text-[10px] text-red-400 bg-red-500/10 px-2 py-1 rounded-lg">
                    Sync error
                  </div>
                )}
                {intg.status === 'connected' && (
                  <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-2">Synced {intg.lastSync}</p>
                )}
                <div className="mt-3 pt-3 border-t dark:border-[#2D3748] border-gray-100 grid grid-cols-2 gap-2">
                  <div>
                    <p className="text-sm font-semibold dark:text-gray-200 text-gray-800">{intg.syncedOrders.toLocaleString()}</p>
                    <p className="text-[9px] dark:text-gray-500 text-gray-400">orders</p>
                  </div>
                  <div>
                    <p className="text-sm font-semibold dark:text-gray-200 text-gray-800">{intg.syncedProducts}</p>
                    <p className="text-[9px] dark:text-gray-500 text-gray-400">products</p>
                  </div>
                </div>
              </button>
            )
          })}
        </div>

        {/* Sync logs */}
        <div className="flex-1 dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden flex flex-col">
          <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Sync Logs</h3>
            <button className="flex items-center gap-1.5 text-xs dark:text-gray-400 text-gray-500 hover:dark:text-gray-200 hover:text-gray-700">
              <RefreshCw size={12} /> Refresh
            </button>
          </div>
          <div className="flex-1 overflow-y-auto font-mono text-xs">
            {MOCK_LOGS.map((log, i) => (
              <div key={i} className={clsx(
                'flex items-start gap-3 px-4 py-2 border-b dark:border-[#2D3748]/50 border-gray-50',
                !log.ok && 'dark:bg-red-500/5 bg-red-50',
              )}>
                <span className="dark:text-gray-600 text-gray-400 text-[10px] shrink-0 mt-0.5">{log.time}</span>
                <span className={clsx('text-[10px] px-1.5 py-0.5 rounded shrink-0',
                  log.integration === 'Shopify' ? 'bg-green-500/15 text-green-400' :
                  log.integration === 'WooCommerce' ? 'bg-purple-500/15 text-purple-400' :
                  log.integration === 'Bol.com' ? 'bg-blue-500/15 text-blue-400' : 'bg-orange-500/15 text-orange-400',
                )}>
                  {log.integration}
                </span>
                <span className={clsx('text-[10px] font-sans', log.ok ? 'dark:text-gray-300 text-gray-700' : 'text-red-400')}>
                  {log.msg}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Right: integration config panel */}
      {selected && (
        <div className="w-80 border-l dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 flex flex-col overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">{selected.name} Config</h2>
            <button onClick={() => setSelectedIntegration(null)} className="text-xs dark:text-gray-500 text-gray-400">✕</button>
          </div>

          {selected.status === 'error' && (
            <div className="mx-4 mt-3 p-3 rounded-lg dark:bg-red-500/10 bg-red-50 border dark:border-red-500/20 border-red-200">
              <p className="text-xs text-red-400 font-medium mb-1">Sync Error</p>
              <p className="text-[10px] text-red-400/80">{selected.error}</p>
              <button className="mt-2 w-full py-1.5 rounded-lg bg-red-500/20 text-red-400 text-xs hover:bg-red-500/30 transition-colors">
                Retry Now
              </button>
            </div>
          )}

          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            <div>
              <label className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">API Key</label>
              <input
                type="password"
                defaultValue="sk_live_••••••••••••••••"
                className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">Store URL</label>
              <input
                type="text"
                defaultValue={`https://mystore.${selected.id}.com`}
                className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">Sync Interval</label>
              <select className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none">
                <option>Every 5 minutes</option>
                <option>Every 15 minutes</option>
                <option>Every hour</option>
                <option>Real-time (webhook)</option>
              </select>
            </div>
            <div className="space-y-2">
              <p className="text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider">Sync Rules</p>
              {['Sync new orders', 'Update stock on sale', 'Push price changes', 'Sync product catalog'].map(rule => (
                <label key={rule} className="flex items-center gap-2 cursor-pointer">
                  <input type="checkbox" defaultChecked className="rounded accent-blue-600 w-3 h-3" />
                  <span className="text-xs dark:text-gray-300 text-gray-700">{rule}</span>
                </label>
              ))}
            </div>
          </div>

          <div className="p-4 border-t dark:border-[#2D3748] border-gray-200 space-y-2">
            <button className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors flex items-center justify-center gap-2">
              <Settings size={12} /> Save Configuration
            </button>
            <button className="w-full py-2 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 text-xs hover:dark:bg-[#2D3748] hover:bg-gray-100 transition-colors flex items-center justify-center gap-2">
              <ExternalLink size={12} /> Open in {selected.name}
            </button>
          </div>
        </div>
      )}

      {/* Add new integration CTA (when none selected) */}
      {!selected && (
        <div className="w-72 border-l dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 flex flex-col items-center justify-center p-6 text-center">
          <div className="w-12 h-12 rounded-xl dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 flex items-center justify-center mb-3">
            <Plug size={20} className="dark:text-gray-500 text-gray-400" />
          </div>
          <p className="text-sm font-medium dark:text-gray-300 text-gray-700 mb-1">Add Integration</p>
          <p className="text-xs dark:text-gray-500 text-gray-400 mb-4">Connect more channels to sync orders and inventory</p>
          <button className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors">
            Browse Integrations
          </button>
        </div>
      )}
    </div>
  )
}
