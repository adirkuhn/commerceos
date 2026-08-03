import { useState } from 'react'
import {
  Building2, Package, Truck, Globe, Zap, Shield, Bell, CreditCard, Plug,
  ChevronRight, Check, AlertTriangle,
} from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useConnections } from '../integrations/hooks/useConnections'
import WooCommerceSyncSettingsPage from '../integrations/WooCommerceSyncSettingsPage'
import type { Connection } from '../integrations/types'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'

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

        {!['company', 'integrations', 'inventory', 'localization', 'automation', 'security', 'notifications'].includes(activeSection) && (
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
      await save({ default_currency: defaultCurrency, timezone, date_format: dateFormat, language })
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
