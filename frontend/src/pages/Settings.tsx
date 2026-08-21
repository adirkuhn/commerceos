import { useState } from 'react'
import {
  Building2, Package, Truck, Globe, Zap, Shield, Bell, CreditCard, Plug,
  ChevronRight, Check, AlertTriangle, Bot, X, Percent, Trash2, TrendingUp,
} from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useConnections } from '../integrations/hooks/useConnections'
import { useAIConnection } from '../integrations/hooks/useAIConnection'
import WooCommerceSyncSettingsPage from '../integrations/WooCommerceSyncSettingsPage'
import type { Connection, AIProvider } from '../integrations/types'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import type { TenantSettings, TenantSettingsInput } from '../settings/types'
import { useTaxClasses } from '../tax/hooks/useTaxClasses'
import { useTaxRates } from '../tax/hooks/useTaxRates'
import type { TaxClass } from '../tax/types'

// toInput picks just the persisted fields off a fetched TenantSettings so a
// section that only edits some of them (Localization edits 4, Tax edits 2)
// can still send a full snapshot -- the backend replaces the whole row on
// every PATCH (see UpdateTenantSettingsUseCase), so omitting a field here
// would silently wipe it back to empty/false instead of leaving it alone.
function toInput(s: TenantSettings | null): TenantSettingsInput {
  if (!s) return {}
  return {
    default_currency: s.default_currency, timezone: s.timezone, date_format: s.date_format, language: s.language,
    prices_include_tax: s.prices_include_tax, default_tax_country: s.default_tax_country,
    inventory_sync_rules_enabled: s.inventory_sync_rules_enabled, batch_expiry_threshold_days: s.batch_expiry_threshold_days,
    default_target_markup_basis_points: s.default_target_markup_basis_points,
  }
}

interface SettingSection {
  id: string
  label: string
  icon: React.ElementType
  description: string
}

const SECTIONS: SettingSection[] = [
  { id: 'company', label: 'Company', icon: Building2, description: 'Business name, address, tax IDs' },
  { id: 'integrations', label: 'Integrations', icon: Plug, description: 'Connect and configure external services' },
  { id: 'inventory', label: 'Inventory Rules', icon: Package, description: 'FEFO, min stock, reorder points' },
  { id: 'shipping', label: 'Shipping Rules', icon: Truck, description: 'Default carriers, label format, zones' },
  { id: 'localization', label: 'Localization', icon: Globe, description: 'Currency, timezone, language' },
  { id: 'tax', label: 'Tax', icon: Percent, description: 'Tax classes and rates' },
  { id: 'pricing', label: 'Pricing', icon: TrendingUp, description: 'Default target markup for recommended prices' },
  { id: 'automation', label: 'Automation', icon: Zap, description: 'Trigger-based rules and workflows' },
  { id: 'security', label: 'Security', icon: Shield, description: 'SSO, 2FA, IP restrictions' },
  { id: 'notifications', label: 'Notifications', icon: Bell, description: 'Email, webhook, Slack alerts' },
  { id: 'billing', label: 'Billing', icon: CreditCard, description: 'Plan, invoices, usage' },
]

type SectionId = typeof SECTIONS[number]['id']

const AUTOMATIONS = [
  { id: '1', name: 'Auto-reorder when stock < min', active: true, trigger: 'Stock below minimum', action: 'Create purchase suggestion' },
  { id: '2', name: 'Alert on expiry < 7 days', active: true, trigger: 'Product expiry within 7 days', action: 'Send email alert + dashboard badge' },
  { id: '3', name: 'Auto-select carrier by weight', active: true, trigger: 'Order created', action: 'Apply carrier rule' },
  { id: '4', name: 'Sync price changes to channels', active: false, trigger: 'Product price updated', action: 'Push to all integrations' },
  { id: '5', name: 'Reserve stock on order', active: true, trigger: 'New order received', action: 'Reserve inventory' },
]

export default function Settings() {
  const [activeSection, setActiveSection] = useState<SectionId>('company')
  const [configuringConnection, setConfiguringConnection] = useState<Connection | null>(null)

  if (configuringConnection) {
    return <WooCommerceSyncSettingsPage connection={configuringConnection} onClose={() => setConfiguringConnection(null)} />
  }

  return (
    <div className="flex h-full overflow-hidden">
      {/* Sidebar */}
      <div className="w-56 border-r dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 flex flex-col overflow-y-auto">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <p className="text-xs font-semibold dark:text-gray-200 text-gray-800">Settings</p>
        </div>
        <nav className="flex-1 py-2 px-2">
          {SECTIONS.map(s => {
            const Icon = s.icon
            const active = activeSection === s.id
            return (
              <button
                key={s.id}
                onClick={() => setActiveSection(s.id)}
                className={clsx(
                  'w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left mb-0.5 transition-colors',
                  active
                    ? 'dark:bg-blue-600/15 bg-blue-50 text-blue-400'
                    : 'dark:text-gray-400 text-gray-600 hover:dark:bg-[#1F2937] hover:bg-white',
                )}
              >
                <Icon size={14} className={active ? 'text-blue-400' : ''} />
                <span className="text-xs font-medium flex-1">{s.label}</span>
                {active && <ChevronRight size={12} className="text-blue-400" />}
              </button>
            )
          })}
        </nav>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-6">
        {activeSection === 'company' && (
          <div className="max-w-2xl space-y-6">
            <div>
              <h2 className="text-base font-semibold dark:text-gray-100 text-gray-900 mb-1">Company Settings</h2>
              <p className="text-xs dark:text-gray-500 text-gray-400">Business information used on invoices, labels, and reports</p>
            </div>
            <div className="space-y-4">
              {[
                ['Company Name', 'OmniStock Operations BV', 'text'],
                ['VAT Number', 'NL123456789B01', 'text'],
                ['Chamber of Commerce', '12345678', 'text'],
                ['Business Email', 'ops@omnistock.io', 'email'],
                ['Phone', '+31 20 123 4567', 'tel'],
              ].map(([label, defaultValue, type]) => (
                <div key={label}>
                  <label className="block text-xs font-medium dark:text-gray-300 text-gray-700 mb-1.5">{label}</label>
                  <input
                    type={type}
                    defaultValue={defaultValue}
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
              ))}
            </div>
            <button className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700">
              Save Changes
            </button>
          </div>
        )}

        {activeSection === 'integrations' && <IntegrationsSettingsSection onConfigure={setConfiguringConnection} />}

        {activeSection === 'inventory' && (
          <div className="max-w-2xl space-y-6">
            <div>
              <h2 className="text-base font-semibold dark:text-gray-100 text-gray-900 mb-1">Inventory Rules</h2>
              <p className="text-xs dark:text-gray-500 text-gray-400">Configure stock management behavior</p>
            </div>

            <InventorySyncRulesSection />

            <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
              <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
                <h3 className="text-xs font-semibold dark:text-gray-200 text-gray-800">FEFO Settings</h3>
                <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">First Expire, First Out — for perishable products</p>
              </div>
              <div className="p-4 space-y-3">
                {[
                  ['Enable FEFO globally', true],
                  ['Alert when product expires within 7 days', true],
                  ['Auto-flag expired stock for disposal', true],
                  ['Block picking of expired stock', false],
                ].map(([label, defaultChecked]) => (
                  <label key={label as string} className="flex items-center gap-3 cursor-pointer">
                    <input type="checkbox" defaultChecked={defaultChecked as boolean} className="rounded accent-blue-600 w-3.5 h-3.5" />
                    <span className="text-xs dark:text-gray-300 text-gray-700">{label as string}</span>
                  </label>
                ))}
              </div>
            </div>

            <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
              <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
                <h3 className="text-xs font-semibold dark:text-gray-200 text-gray-800">Stock Thresholds</h3>
              </div>
              <div className="p-4 space-y-3">
                {[
                  ['Global minimum stock (units)', '20'],
                  ['Critical stock threshold (%)', '15'],
                  ['Overstock threshold (%)', '200'],
                ].map(([label, val]) => (
                  <div key={label} className="flex items-center gap-4">
                    <label className="text-xs dark:text-gray-300 text-gray-700 flex-1">{label}</label>
                    <input
                      type="number"
                      defaultValue={val}
                      className="w-24 px-2 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-gray-50 border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 outline-none focus:ring-1 focus:ring-blue-500 text-right"
                    />
                  </div>
                ))}
              </div>
            </div>
            <button className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700">Save Changes</button>
          </div>
        )}

        {activeSection === 'localization' && <LocalizationSettingsSection />}
        {activeSection === 'tax' && (
          <div className="max-w-2xl space-y-6">
            <TaxSettingsSection />
          </div>
        )}

        {activeSection === 'pricing' && (
          <div className="max-w-2xl space-y-6">
            <div>
              <h2 className="text-base font-semibold dark:text-gray-100 text-gray-900 mb-1">Pricing</h2>
              <p className="text-xs dark:text-gray-500 text-gray-400">Default target markup used to compute recommended prices</p>
            </div>
            <PricingMarginSection />
          </div>
        )}

        {activeSection === 'automation' && (
          <div className="max-w-2xl space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-base font-semibold dark:text-gray-100 text-gray-900 mb-1">Automation Rules</h2>
                <p className="text-xs dark:text-gray-500 text-gray-400">Trigger-based workflows that run automatically</p>
              </div>
              <button className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700">
                + New Rule
              </button>
            </div>

            <div className="space-y-3">
              {AUTOMATIONS.map(auto => (
                <div key={auto.id} className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <p className="text-xs font-medium dark:text-gray-100 text-gray-900">{auto.name}</p>
                        {auto.active
                          ? <span className="px-1.5 py-0.5 rounded-full text-[9px] font-medium bg-green-500/15 text-green-400">active</span>
                          : <span className="px-1.5 py-0.5 rounded-full text-[9px] font-medium dark:bg-gray-500/15 bg-gray-100 dark:text-gray-400 text-gray-500">inactive</span>
                        }
                      </div>
                      <div className="flex items-center gap-2 text-[10px]">
                        <span className="dark:text-gray-500 text-gray-400">When:</span>
                        <span className="dark:bg-[#2D3748] bg-gray-100 dark:text-gray-300 text-gray-600 px-2 py-0.5 rounded">{auto.trigger}</span>
                        <ChevronRight size={10} className="dark:text-gray-600 text-gray-400" />
                        <span className="dark:text-gray-500 text-gray-400">Then:</span>
                        <span className="dark:bg-blue-500/10 bg-blue-50 text-blue-400 px-2 py-0.5 rounded">{auto.action}</span>
                      </div>
                    </div>
                    <label className="relative flex cursor-pointer items-center shrink-0">
                      <input type="checkbox" defaultChecked={auto.active} className="sr-only peer" />
                      <div className="w-8 h-4 dark:bg-[#2D3748] bg-gray-300 rounded-full peer peer-checked:bg-blue-600 transition-colors" />
                      <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
                    </label>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {activeSection === 'security' && (
          <div className="max-w-2xl space-y-6">
            <div>
              <h2 className="text-base font-semibold dark:text-gray-100 text-gray-900 mb-1">Security</h2>
              <p className="text-xs dark:text-gray-500 text-gray-400">Authentication and access control</p>
            </div>
            <div className="dark:bg-amber-500/5 bg-amber-50 border dark:border-amber-500/20 border-amber-200 rounded-xl p-4 flex items-start gap-3">
              <AlertTriangle size={15} className="text-amber-400 mt-0.5 shrink-0" />
              <div>
                <p className="text-xs font-medium text-amber-400">2FA not enabled for all users</p>
                <p className="text-[10px] text-amber-400/70 mt-0.5">3 users have not enabled two-factor authentication.</p>
              </div>
            </div>
            <div className="space-y-3">
              {[
                { label: 'Enforce 2FA for all users', checked: false },
                { label: 'Require SSO login', checked: false },
                { label: 'Session timeout after 8h', checked: true },
                { label: 'IP allowlist restriction', checked: false },
                { label: 'Audit log all actions', checked: true },
              ].map(item => (
                <label key={item.label} className="flex items-center gap-3 cursor-pointer dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 rounded-xl p-3.5">
                  <input type="checkbox" defaultChecked={item.checked} className="rounded accent-blue-600 w-3.5 h-3.5 shrink-0" />
                  <span className="text-xs dark:text-gray-300 text-gray-700">{item.label}</span>
                  {item.checked && <Check size={12} className="text-green-400 ml-auto" />}
                </label>
              ))}
            </div>
          </div>
        )}

        {activeSection === 'notifications' && (
          <div className="max-w-2xl space-y-6">
            <div>
              <h2 className="text-base font-semibold dark:text-gray-100 text-gray-900 mb-1">Notifications</h2>
              <p className="text-xs dark:text-gray-500 text-gray-400">Configure alerts across channels</p>
            </div>
            {[
              { channel: 'Email', events: ['Critical stock alert', 'Expiry alert', 'Integration failure', 'New order'] },
              { channel: 'Webhook', events: ['Order status change', 'Stock adjustment', 'Sync complete'] },
              { channel: 'Slack', events: ['Critical stock alert', 'Integration failure'] },
            ].map(section => (
              <div key={section.channel} className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
                <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200 flex items-center justify-between">
                  <p className="text-xs font-semibold dark:text-gray-200 text-gray-800">{section.channel}</p>
                  <span className="text-[10px] text-green-400 bg-green-500/15 px-2 py-0.5 rounded-full">configured</span>
                </div>
                <div className="p-4 space-y-2">
                  {section.events.map(ev => (
                    <label key={ev} className="flex items-center gap-3 cursor-pointer">
                      <input type="checkbox" defaultChecked className="rounded accent-blue-600 w-3 h-3" />
                      <span className="text-xs dark:text-gray-300 text-gray-700">{ev}</span>
                    </label>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}

        {!['company', 'integrations', 'inventory', 'localization', 'automation', 'security', 'notifications', 'tax', 'pricing'].includes(activeSection) && (
          <div className="max-w-2xl">
            <h2 className="text-base font-semibold dark:text-gray-100 text-gray-900 mb-1 capitalize">{activeSection}</h2>
            <p className="text-xs dark:text-gray-500 text-gray-400">
              {SECTIONS.find(s => s.id === activeSection)?.description}
            </p>
            <div className="mt-8 flex flex-col items-center text-center">
              <div className="w-12 h-12 dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 rounded-xl flex items-center justify-center mb-3">
                {(() => { const Icon = SECTIONS.find(s => s.id === activeSection)?.icon ?? Zap; return <Icon size={20} className="dark:text-gray-500 text-gray-400" /> })()}
              </div>
              <p className="text-xs dark:text-gray-500 text-gray-400">Configuration coming soon</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

// ── Integrations settings section ─────────────────────────────────────────────

const STATUS_STYLES: Record<Connection['status'], string> = {
  connected: 'bg-green-500/15 text-green-400',
  error: 'bg-red-500/15 text-red-400',
}

function IntegrationsSettingsSection({ onConfigure }: { onConfigure: (connection: Connection) => void }) {
  const { token } = useAuth()
  const { connections, loading, error } = useConnections(token ?? '')

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-base font-semibold dark:text-gray-100 text-gray-900 mb-1">Integrations</h2>
        <p className="text-xs dark:text-gray-500 text-gray-400">Connected external services. Configure sync schedules for each.</p>
      </div>

      <AIIntegrationCard />

      {loading && <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
      {error && <p className="text-xs text-red-400">{error}</p>}

      {!loading && connections.length === 0 && (
        <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-6 text-center">
          <p className="text-xs dark:text-gray-500 text-gray-400">No integrations connected yet.</p>
        </div>
      )}

      <div className="space-y-3">
        {connections.map(conn => (
          <div key={conn.id} className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 flex items-center justify-between gap-4">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <p className="text-xs font-medium dark:text-gray-200 text-gray-800 capitalize">{conn.provider}</p>
                <span className={clsx('px-1.5 py-0.5 rounded-full text-[9px] font-medium capitalize', STATUS_STYLES[conn.status])}>
                  {conn.status}
                </span>
              </div>
              <p className="text-[10px] dark:text-gray-500 text-gray-400 truncate mt-0.5">{conn.store_url}</p>
            </div>
            {conn.provider === 'woocommerce' ? (
              <button
                onClick={() => onConfigure(conn)}
                className="px-3 py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 dark:text-gray-300 text-gray-600 text-xs font-medium hover:dark:bg-[#2D3748] hover:bg-gray-200 transition-colors shrink-0"
              >
                Configure
              </button>
            ) : (
              <span className="text-[10px] dark:text-gray-600 text-gray-400 shrink-0">Coming soon</span>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

const AI_PROVIDER_LABELS: Record<AIProvider, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic (Claude)',
  gemini: 'Google Gemini',
  ollama: 'Ollama (self-hosted)',
}

// AI is a separate integration shape from Connection (WooCommerce etc.) --
// one row per tenant, no store URL/OAuth, just a provider + credential --
// so it gets its own card and configure flow rather than joining the
// Connection list above.
function AIIntegrationCard() {
  const { token } = useAuth()
  const { connection, loading, error, connect, disconnect } = useAIConnection(token ?? '')
  const [configuring, setConfiguring] = useState(false)
  const [provider, setProvider] = useState<AIProvider>('openai')
  const [apiKey, setApiKey] = useState('')
  const [baseUrl, setBaseUrl] = useState('')
  const [model, setModel] = useState('')
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  const openConfigure = () => {
    setProvider(connection?.provider ?? 'openai')
    setApiKey('')
    setBaseUrl(connection?.base_url ?? '')
    setModel(connection?.model ?? '')
    setFormError('')
    setConfiguring(true)
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setFormError('')
    try {
      await connect({ provider, apiKey, baseUrl, model })
      setConfiguring(false)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to connect')
    } finally {
      setSaving(false)
    }
  }

  const handleDisconnect = async () => {
    if (!window.confirm('Disconnect AI provider? Procurement PDF document parsing will stop working until reconfigured.')) return
    await disconnect()
  }

  return (
    <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0 flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-cyan-600 flex items-center justify-center text-white shrink-0">
            <Bot size={14} />
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <p className="text-xs font-medium dark:text-gray-200 text-gray-800">AI</p>
              {connection && (
                <span className={clsx('px-1.5 py-0.5 rounded-full text-[9px] font-medium capitalize', STATUS_STYLES[connection.status])}>
                  {connection.status}
                </span>
              )}
            </div>
            <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">
              {loading ? 'Loading…' : connection
                ? AI_PROVIDER_LABELS[connection.provider]
                : 'Used for procurement document parsing (PDF invoices). Not configured.'}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <button
            onClick={openConfigure}
            className="px-3 py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 dark:text-gray-300 text-gray-600 text-xs font-medium hover:dark:bg-[#2D3748] hover:bg-gray-200 transition-colors"
          >
            {connection ? 'Configure' : 'Connect'}
          </button>
          {connection && (
            <button
              onClick={handleDisconnect}
              className="px-3 py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-xs font-medium dark:text-gray-400 text-gray-500 hover:text-red-400 transition-colors"
            >
              Disconnect
            </button>
          )}
        </div>
      </div>
      {error && <p className="mt-2 text-[10px] text-red-400">{error}</p>}

      {configuring && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={() => setConfiguring(false)} />
          <div className="relative w-full max-w-sm dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Configure AI</h3>
              <button onClick={() => setConfiguring(false)} aria-label="Close">
                <X size={14} className="dark:text-gray-500 text-gray-400" />
              </button>
            </div>
            <form onSubmit={handleSave} className="space-y-3">
              <div>
                <label htmlFor="ai-provider" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                  Provider
                </label>
                <select
                  id="ai-provider"
                  value={provider}
                  onChange={e => setProvider(e.target.value as AIProvider)}
                  className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                >
                  {(Object.keys(AI_PROVIDER_LABELS) as AIProvider[]).map(p => (
                    <option key={p} value={p}>{AI_PROVIDER_LABELS[p]}</option>
                  ))}
                </select>
              </div>
              {provider !== 'ollama' && (
                <div>
                  <label htmlFor="ai-api-key" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                    API Key
                  </label>
                  <input
                    id="ai-api-key"
                    type="password"
                    required
                    autoFocus
                    value={apiKey}
                    onChange={e => setApiKey(e.target.value)}
                    placeholder="sk-..."
                    className="w-full px-3 py-2 text-xs font-mono rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
              )}
              <div>
                <label htmlFor="ai-base-url" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                  {provider === 'ollama' ? 'Server URL' : 'Base URL (optional)'}
                </label>
                <input
                  id="ai-base-url"
                  type="url"
                  required={provider === 'ollama'}
                  autoFocus={provider === 'ollama'}
                  value={baseUrl}
                  onChange={e => setBaseUrl(e.target.value)}
                  placeholder={provider === 'ollama' ? 'http://localhost:11434' : 'https://...'}
                  className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div>
                <label htmlFor="ai-model" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                  Model (optional)
                </label>
                <input
                  id="ai-model"
                  type="text"
                  value={model}
                  onChange={e => setModel(e.target.value)}
                  placeholder="Uses the provider's default"
                  className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              {formError && (
                <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
              <button
                type="submit"
                disabled={saving}
                className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                {saving ? 'Saving…' : 'Save'}
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

// ── Localization settings section ─────────────────────────────────────────────

const CURRENCY_OPTIONS = [
  { value: 'EUR', label: 'EUR — Euro' },
  { value: 'USD', label: 'USD — US Dollar' },
  { value: 'GBP', label: 'GBP — British Pound' },
  { value: 'CHF', label: 'CHF — Swiss Franc' },
]
const TIMEZONE_OPTIONS = [
  { value: 'Europe/Amsterdam', label: 'Europe/Amsterdam (UTC+2)' },
  { value: 'Europe/London', label: 'Europe/London (UTC+1)' },
  { value: 'Europe/Paris', label: 'Europe/Paris (UTC+2)' },
  { value: 'America/New_York', label: 'America/New_York (UTC-4)' },
]
const DATE_FORMAT_OPTIONS = [
  { value: 'DD/MM/YYYY', label: 'DD/MM/YYYY' },
  { value: 'MM/DD/YYYY', label: 'MM/DD/YYYY' },
  { value: 'YYYY-MM-DD', label: 'YYYY-MM-DD' },
]
const LANGUAGE_OPTIONS = [
  { value: 'en', label: 'English (EN)' },
  { value: 'nl', label: 'Dutch (NL)' },
  { value: 'de', label: 'German (DE)' },
  { value: 'fr', label: 'French (FR)' },
]

function InventorySyncRulesSection() {
  const { token } = useAuth()
  const { settings, save } = useTenantSettings(token ?? '')

  const [enabled, setEnabled] = useState(false)
  const [thresholdDays, setThresholdDays] = useState(0)
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [saved, setSaved] = useState(false)

  if (!hydrated && settings) {
    setEnabled(settings.inventory_sync_rules_enabled)
    setThresholdDays(settings.batch_expiry_threshold_days)
    setHydrated(true)
  }

  async function handleSave() {
    setSaving(true)
    setSaveError('')
    setSaved(false)
    try {
      await save({ ...toInput(settings), inventory_sync_rules_enabled: enabled, batch_expiry_threshold_days: thresholdDays })
      setSaved(true)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save inventory rules')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
      <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Sync & expiry rules</h3>
      <div className="flex items-center justify-between">
        <div>
          <span className="text-xs dark:text-gray-300 text-gray-700 block">Enable inventory sync rules</span>
          <span className="text-[10px] dark:text-gray-500 text-gray-400">
            Controls what quantity syncs to channels: no batches → ledger total · batches with no expiry → summed together · batches with an expiry → only the nearest-expiring batch's quantity, so channels sell through the oldest stock first.
          </span>
        </div>
        <label className="relative flex items-center shrink-0 cursor-pointer">
          <input
            type="checkbox"
            role="switch"
            aria-label="Enable inventory sync rules"
            checked={enabled}
            onChange={e => { setEnabled(e.target.checked); setSaved(false) }}
            className="sr-only peer"
          />
          <div className="w-8 h-4 dark:bg-[#2D3748] bg-gray-300 rounded-full peer peer-checked:bg-blue-600 transition-colors" />
          <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
        </label>
      </div>
      <div className="max-w-[220px]">
        <label htmlFor="batch-expiry-threshold-days" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
          Auto-expire batches after (days)
        </label>
        <input
          id="batch-expiry-threshold-days"
          type="number"
          min={0}
          value={thresholdDays}
          onChange={e => { setThresholdDays(parseInt(e.target.value, 10) || 0); setSaved(false) }}
          className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
        />
      </div>
      {saveError && <p className="text-xs text-red-400">{saveError}</p>}
      {saved && <p className="text-xs text-green-400">Saved.</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-3 py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-xs font-medium dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-200 disabled:opacity-50 transition-colors"
      >
        {saving ? 'Saving…' : 'Save inventory rules'}
      </button>
    </div>
  )
}

function PricingMarginSection() {
  const { token } = useAuth()
  const { settings, save } = useTenantSettings(token ?? '')

  const [markupPercent, setMarkupPercent] = useState('')
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [saved, setSaved] = useState(false)

  if (!hydrated && settings) {
    setMarkupPercent(settings.default_target_markup_basis_points ? (settings.default_target_markup_basis_points / 100).toString() : '')
    setHydrated(true)
  }

  async function handleSave() {
    setSaveError('')
    setSaved(false)
    const basisPoints = markupPercent.trim() === '' ? 0 : Math.round(parseFloat(markupPercent) * 100)
    if (basisPoints < 0) {
      setSaveError('Target markup cannot be negative.')
      return
    }
    setSaving(true)
    try {
      await save({ ...toInput(settings), default_target_markup_basis_points: basisPoints })
      setSaved(true)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save pricing settings')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
      <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Pricing & markup</h3>
      <div className="max-w-[220px]">
        <label htmlFor="default-target-markup" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
          Default target markup (%)
        </label>
        <input
          id="default-target-markup"
          type="number" min={0} step={0.01}
          value={markupPercent}
          onChange={e => { setMarkupPercent(e.target.value); setSaved(false) }}
          placeholder="e.g. 30"
          className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
        />
        <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-1">
          Markup is profit as a % of cost (e.g. 300% markup means selling at 4x cost) -- no upper bound, unlike margin.
          Used to compute each product's recommended price from its cost price, unless its category has its own target markup set.
        </p>
      </div>
      {saveError && <p className="text-xs text-red-400">{saveError}</p>}
      {saved && <p className="text-xs text-green-400">Saved.</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-3 py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-xs font-medium dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-200 disabled:opacity-50 transition-colors"
      >
        {saving ? 'Saving…' : 'Save pricing settings'}
      </button>
    </div>
  )
}

function LocalizationSettingsSection() {
  const { token } = useAuth()
  const { settings, loading, error, save } = useTenantSettings(token ?? '')

  const [defaultCurrency, setDefaultCurrency] = useState(CURRENCY_OPTIONS[0].value)
  const [timezone, setTimezone] = useState(TIMEZONE_OPTIONS[0].value)
  const [dateFormat, setDateFormat] = useState(DATE_FORMAT_OPTIONS[0].value)
  const [language, setLanguage] = useState(LANGUAGE_OPTIONS[0].value)
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [saved, setSaved] = useState(false)

  if (!hydrated && settings) {
    if (settings.default_currency) setDefaultCurrency(settings.default_currency)
    if (settings.timezone) setTimezone(settings.timezone)
    if (settings.date_format) setDateFormat(settings.date_format)
    if (settings.language) setLanguage(settings.language)
    setHydrated(true)
  }

  async function handleSave() {
    setSaving(true)
    setSaveError('')
    setSaved(false)
    try {
      await save({ ...toInput(settings), default_currency: defaultCurrency, timezone, date_format: dateFormat, language })
      setSaved(true)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save settings')
    } finally {
      setSaving(false)
    }
  }

  const fields: { id: string; label: string; value: string; onChange: (v: string) => void; options: { value: string; label: string }[] }[] = [
    { id: 'localization-currency', label: 'Default Currency', value: defaultCurrency, onChange: setDefaultCurrency, options: CURRENCY_OPTIONS },
    { id: 'localization-timezone', label: 'Timezone', value: timezone, onChange: setTimezone, options: TIMEZONE_OPTIONS },
    { id: 'localization-date-format', label: 'Date Format', value: dateFormat, onChange: setDateFormat, options: DATE_FORMAT_OPTIONS },
    { id: 'localization-language', label: 'Language', value: language, onChange: setLanguage, options: LANGUAGE_OPTIONS },
  ]

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-base font-semibold dark:text-gray-100 text-gray-900 mb-1">Localization</h2>
        <p className="text-xs dark:text-gray-500 text-gray-400">Currency, timezone, date formats</p>
      </div>
      {loading && <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
      <div className="space-y-4">
        {fields.map(({ id, label, value, onChange, options }) => (
          <div key={id}>
            <label htmlFor={id} className="block text-xs font-medium dark:text-gray-300 text-gray-700 mb-1.5">{label}</label>
            <select
              id={id}
              value={value}
              onChange={e => { onChange(e.target.value); setSaved(false) }}
              className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 outline-none focus:ring-1 focus:ring-blue-500"
            >
              {options.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
            </select>
          </div>
        ))}
      </div>
      {(error || saveError) && <p className="text-xs text-red-400">{error || saveError}</p>}
      {saved && <p className="text-xs text-green-400">Saved.</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50"
      >
        {saving ? 'Saving…' : 'Save Changes'}
      </button>
    </div>
  )
}

interface TaxClassFormState {
  name: string
  isDefault: boolean
}

function TaxSettingsSection() {
  const { token } = useAuth()
  const { settings, save } = useTenantSettings(token ?? '')
  const { classes, loading: classesLoading, error: classesError, create, update, remove } = useTaxClasses(token ?? '')
  const { rates, error: ratesError, upsert, remove: removeRate } = useTaxRates(token ?? '')

  const [pricesIncludeTax, setPricesIncludeTax] = useState(true)
  const [defaultTaxCountry, setDefaultTaxCountry] = useState('')
  const [hydrated, setHydrated] = useState(false)
  const [savingDefaults, setSavingDefaults] = useState(false)
  const [defaultsError, setDefaultsError] = useState('')
  const [defaultsSaved, setDefaultsSaved] = useState(false)

  if (!hydrated && settings) {
    setPricesIncludeTax(settings.prices_include_tax)
    setDefaultTaxCountry(settings.default_tax_country)
    setHydrated(true)
  }

  async function handleSaveDefaults() {
    setSavingDefaults(true)
    setDefaultsError('')
    setDefaultsSaved(false)
    try {
      await save({ ...toInput(settings), prices_include_tax: pricesIncludeTax, default_tax_country: defaultTaxCountry })
      setDefaultsSaved(true)
    } catch (err) {
      setDefaultsError(err instanceof Error ? err.message : 'Failed to save tax defaults')
    } finally {
      setSavingDefaults(false)
    }
  }

  const [creatingClass, setCreatingClass] = useState(false)
  const [editingClass, setEditingClass] = useState<TaxClass | null>(null)
  const [classForm, setClassForm] = useState<TaxClassFormState>({ name: '', isDefault: false })
  const [classSaving, setClassSaving] = useState(false)
  const [classFormError, setClassFormError] = useState('')
  const [rowError, setRowError] = useState('')

  const openCreateClass = () => {
    setClassForm({ name: '', isDefault: false })
    setClassFormError('')
    setCreatingClass(true)
  }
  const openEditClass = (c: TaxClass) => {
    setClassForm({ name: c.name, isDefault: c.is_default })
    setClassFormError('')
    setEditingClass(c)
  }
  const closeClassForm = () => {
    setCreatingClass(false)
    setEditingClass(null)
    setClassFormError('')
  }

  async function handleClassSubmit(e: React.FormEvent) {
    e.preventDefault()
    setClassFormError('')
    setClassSaving(true)
    try {
      if (editingClass) {
        await update(editingClass.id, { name: classForm.name, is_default: classForm.isDefault })
      } else {
        await create({ name: classForm.name, is_default: classForm.isDefault })
      }
      closeClassForm()
    } catch (err) {
      setClassFormError(err instanceof Error ? err.message : 'Failed to save tax class')
    } finally {
      setClassSaving(false)
    }
  }

  async function handleDeleteClass(c: TaxClass) {
    if (!window.confirm(`Delete tax class "${c.name}"? This cannot be undone.`)) return
    setRowError('')
    try {
      await remove(c.id)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to delete tax class')
    }
  }

  async function handleAddRate(taxClassId: string, country: string, ratePercent: string) {
    setRowError('')
    try {
      await upsert({ tax_class_id: taxClassId, country, rate_basis_points: Math.round(parseFloat(ratePercent) * 100) })
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to save tax rate')
    }
  }

  async function handleRemoveRate(id: string) {
    setRowError('')
    try {
      await removeRate(id)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to delete tax rate')
    }
  }

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-base font-semibold dark:text-gray-100 text-gray-900 mb-1">Tax</h2>
        <p className="text-xs dark:text-gray-500 text-gray-400">Tax classes and their per-country rates</p>
      </div>

      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
        <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Store defaults</h3>
        <div className="flex items-center justify-between">
          <span className="text-xs dark:text-gray-300 text-gray-700">Prices include tax</span>
          <label className="relative flex items-center shrink-0 cursor-pointer">
            <input
              type="checkbox"
              role="switch"
              aria-label="Prices include tax"
              checked={pricesIncludeTax}
              onChange={e => { setPricesIncludeTax(e.target.checked); setDefaultsSaved(false) }}
              className="sr-only peer"
            />
            <div className="w-8 h-4 dark:bg-[#2D3748] bg-gray-300 rounded-full peer peer-checked:bg-blue-600 transition-colors" />
            <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
          </label>
        </div>
        <div className="max-w-[160px]">
          <label htmlFor="default-tax-country" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
            Default tax country
          </label>
          <input
            id="default-tax-country"
            value={defaultTaxCountry}
            maxLength={2}
            onChange={e => { setDefaultTaxCountry(e.target.value.toUpperCase()); setDefaultsSaved(false) }}
            className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        {defaultsError && <p className="text-xs text-red-400">{defaultsError}</p>}
        {defaultsSaved && <p className="text-xs text-green-400">Saved.</p>}
        <button
          onClick={handleSaveDefaults}
          disabled={savingDefaults}
          className="px-3 py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-xs font-medium dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-200 disabled:opacity-50 transition-colors"
        >
          {savingDefaults ? 'Saving…' : 'Save tax defaults'}
        </button>
      </div>

      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Tax classes</h3>
        <button
          onClick={openCreateClass}
          className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
        >
          Add tax class
        </button>
      </div>
      {classesError && <p className="text-xs text-red-400">{classesError}</p>}
      {ratesError && <p className="text-xs text-red-400">{ratesError}</p>}
      {rowError && <p className="text-xs text-red-400">{rowError}</p>}
      {classesLoading && <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>}

      <div className="space-y-3">
        {classes.map(c => (
          <div key={c.id} className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium dark:text-gray-100 text-gray-900">{c.name}</span>
                {c.is_default && (
                  <span className="text-[10px] px-2 py-0.5 rounded-full dark:bg-blue-500/10 bg-blue-50 text-blue-500">Default</span>
                )}
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => openEditClass(c)}
                  className="py-1 px-2 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-blue-400 transition-colors"
                >
                  Edit
                </button>
                <button
                  onClick={() => handleDeleteClass(c)}
                  className="py-1 px-2 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-red-400 transition-colors"
                >
                  Delete
                </button>
              </div>
            </div>
            <div className="mt-3 space-y-1.5">
              {rates.filter(r => r.tax_class_id === c.id).map(r => (
                <div key={r.id} className="flex items-center justify-between text-xs dark:text-gray-300 text-gray-700">
                  <span>{r.country}</span>
                  <span>{(r.rate_basis_points / 100).toFixed(2)}%</span>
                  <button
                    onClick={() => handleRemoveRate(r.id)}
                    className="text-[10px] dark:text-gray-500 text-gray-400 hover:text-red-400 transition-colors flex items-center gap-1"
                  >
                    <Trash2 size={11} /> Remove rate
                  </button>
                </div>
              ))}
            </div>
            <TaxRateAddForm taxClassId={c.id} onAdd={handleAddRate} />
          </div>
        ))}
      </div>

      {(creatingClass || editingClass) && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={closeClassForm} />
          <div className="relative w-full max-w-sm dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">
                {editingClass ? 'Edit tax class' : 'Add tax class'}
              </h3>
              <button onClick={closeClassForm} aria-label="Close">
                <X size={14} className="dark:text-gray-500 text-gray-400" />
              </button>
            </div>
            <form onSubmit={handleClassSubmit} className="space-y-3">
              <div>
                <label htmlFor="tax-class-name" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                  Name
                </label>
                <input
                  id="tax-class-name"
                  type="text"
                  required
                  autoFocus
                  value={classForm.name}
                  onChange={e => setClassForm(f => ({ ...f, name: e.target.value }))}
                  className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div className="flex items-center justify-between">
                <span className="text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider">Default tax class</span>
                <label className="relative flex items-center shrink-0 cursor-pointer">
                  <input
                    type="checkbox"
                    role="switch"
                    aria-label="Default tax class"
                    checked={classForm.isDefault}
                    onChange={e => setClassForm(f => ({ ...f, isDefault: e.target.checked }))}
                    className="sr-only peer"
                  />
                  <div className="w-8 h-4 dark:bg-[#2D3748] bg-gray-300 rounded-full peer peer-checked:bg-blue-600 transition-colors" />
                  <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
                </label>
              </div>
              {classFormError && (
                <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{classFormError}</p>
              )}
              <button
                type="submit"
                disabled={classSaving}
                className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                {classSaving ? 'Saving…' : editingClass ? 'Save changes' : 'Add class'}
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

function TaxRateAddForm({ taxClassId, onAdd }: { taxClassId: string; onAdd: (taxClassId: string, country: string, ratePercent: string) => void }) {
  const [country, setCountry] = useState('')
  const [ratePercent, setRatePercent] = useState('')

  return (
    <form
      onSubmit={e => {
        e.preventDefault()
        onAdd(taxClassId, country, ratePercent)
        setCountry('')
        setRatePercent('')
      }}
      className="mt-3 pt-3 border-t dark:border-[#2D3748] border-gray-200 flex items-end gap-2"
    >
      <div>
        <label htmlFor={`tax-rate-country-${taxClassId}`} className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
          Country
        </label>
        <input
          id={`tax-rate-country-${taxClassId}`}
          value={country}
          maxLength={2}
          onChange={e => setCountry(e.target.value.toUpperCase())}
          className="w-16 px-2 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none"
        />
      </div>
      <div>
        <label htmlFor={`tax-rate-percent-${taxClassId}`} className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
          Rate %
        </label>
        <input
          id={`tax-rate-percent-${taxClassId}`}
          type="number"
          step="0.01"
          value={ratePercent}
          onChange={e => setRatePercent(e.target.value)}
          className="w-20 px-2 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none"
        />
      </div>
      <button
        type="submit"
        className="px-3 py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-xs font-medium dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-200 transition-colors"
      >
        Add rate
      </button>
    </form>
  )
}
