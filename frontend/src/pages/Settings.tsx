import { useState, useEffect } from 'react'
import {
  Building2, Package, Truck, Globe, Zap, Shield, Bell, CreditCard, Plug,
  ChevronRight, Check, AlertTriangle, Bot, X, Percent, Trash2, TrendingUp,
} from 'lucide-react'
import clsx from 'clsx'
import { useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { subPathSegments } from '../routes'
import { useStackNavigation } from '../routing/useStackNavigation'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useInitOnce } from '../routing/useInitOnce'
import { ConnectionRoute } from '../integrations/ConnectionRoute'
import { useConnections } from '../integrations/hooks/useConnections'
import { useAIConnection } from '../integrations/hooks/useAIConnection'
import WooCommerceSyncSettingsPage from '../integrations/WooCommerceSyncSettingsPage'
import * as integrationsApi from '../integrations/api'
import type { Connection, AIProvider, SalesChannel } from '../integrations/types'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import * as settingsApi from '../settings/api'
import type { TenantSettings, TenantSettingsInput, ApiKey } from '../settings/types'
import { CURRENCY_OPTIONS } from '../settings/currencyOptions'
import { useTaxClasses } from '../tax/hooks/useTaxClasses'
import { useTaxRates } from '../tax/hooks/useTaxRates'
import type { TaxClass } from '../tax/types'
import { useExchangeRates } from '../fx/hooks/useExchangeRates'

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
    bundle_confidence_threshold: s.bundle_confidence_threshold,
    recommendations_min_support: s.recommendations_min_support,
    recommendations_lookback_days: s.recommendations_lookback_days,
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

const SETTINGS_PATH = '/settings'
const TAX_CLASSES_PATH = `${SETTINGS_PATH}/tax/classes`
const DEFAULT_SECTION: SectionId = 'company'
const INTEGRATIONS_SECTION_PATH = `${SETTINGS_PATH}/integrations`

function sectionPath(id: SectionId): string {
  return id === DEFAULT_SECTION ? SETTINGS_PATH : `${SETTINGS_PATH}/${id}`
}

// Each section is a URL (/settings/tax, ...), as is a connection's sync
// settings page (/settings/integrations/<connectionId>) and the tax class
// editor (/settings/tax/classes/<id|new>) -- refresh and Back keep the
// user where they were. An unknown section shows Company.
export default function Settings() {
  const location = useLocation()
  const navigate = useNavigate()
  const { open, exitTo } = useStackNavigation()
  const [sectionSegment, connectionId] = subPathSegments(location.pathname, SETTINGS_PATH)
  const activeSection: SectionId = SECTIONS.some(s => s.id === sectionSegment) ? sectionSegment : DEFAULT_SECTION
  const setActiveSection = (id: SectionId) => {
    if (id !== activeSection) navigate(sectionPath(id))
  }

  if (activeSection === 'integrations' && connectionId) {
    const back = () => exitTo(INTEGRATIONS_SECTION_PATH)
    return (
      <ConnectionRoute connectionId={connectionId} onBack={back}>
        {connection => <WooCommerceSyncSettingsPage connection={connection} onClose={back} />}
      </ConnectionRoute>
    )
  }

  return (
    <div className="flex h-full overflow-hidden">
      {/* Sidebar */}
      <div className="w-56 border-r border-line bg-sunken flex flex-col overflow-y-auto">
        <div className="px-4 py-3 border-b border-line">
          <p className="text-xs font-semibold text-ink">Settings</p>
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
                    ? 'dark:bg-blue-600/15 bg-blue-50 text-accent'
                    : 'text-ink-3 hover:bg-raised',
                )}
              >
                <Icon size={14} className={active ? 'text-accent' : ''} />
                <span className="text-xs font-medium flex-1">{s.label}</span>
                {active && <ChevronRight size={12} className="text-accent" />}
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
              <h2 className="text-base font-semibold text-ink mb-1">Company Settings</h2>
              <p className="text-xs text-ink-4">Business information used on invoices, labels, and reports</p>
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
                  <label className="block text-xs font-medium text-ink-2 mb-1.5">{label}</label>
                  <input
                    type={type}
                    defaultValue={defaultValue}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
              ))}
            </div>
            <button className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700">
              Save Changes
            </button>
          </div>
        )}

        {activeSection === 'integrations' && <IntegrationsSettingsSection onConfigure={c => open(`${INTEGRATIONS_SECTION_PATH}/${encodeURIComponent(c.id)}`)} />}

        {activeSection === 'inventory' && (
          <div className="max-w-2xl space-y-6">
            <div>
              <h2 className="text-base font-semibold text-ink mb-1">Inventory Rules</h2>
              <p className="text-xs text-ink-4">Configure stock management behavior</p>
            </div>

            <InventorySyncRulesSection />

            <div className="bg-raised rounded-xl border border-line overflow-hidden">
              <div className="px-4 py-3 border-b border-line">
                <h3 className="text-xs font-semibold text-ink">FEFO Settings</h3>
                <p className="text-2xs text-ink-4 mt-0.5">First Expire, First Out — for perishable products</p>
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
                    <span className="text-xs text-ink-2">{label as string}</span>
                  </label>
                ))}
              </div>
            </div>

            <div className="bg-raised rounded-xl border border-line overflow-hidden">
              <div className="px-4 py-3 border-b border-line">
                <h3 className="text-xs font-semibold text-ink">Stock Thresholds</h3>
              </div>
              <div className="p-4 space-y-3">
                {[
                  ['Global minimum stock (units)', '20'],
                  ['Critical stock threshold (%)', '15'],
                  ['Overstock threshold (%)', '200'],
                ].map(([label, val]) => (
                  <div key={label} className="flex items-center gap-4">
                    <label className="text-xs text-ink-2 flex-1">{label}</label>
                    <input
                      type="number"
                      defaultValue={val}
                      className="w-24 px-2 py-1.5 text-xs rounded-lg bg-sunken border border-line text-ink outline-none focus:ring-1 focus:ring-blue-500 text-right"
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
              <h2 className="text-base font-semibold text-ink mb-1">Pricing</h2>
              <p className="text-xs text-ink-4">Default target markup used to compute recommended prices</p>
            </div>
            <PricingMarginSection />
          </div>
        )}

        {activeSection === 'automation' && (
          <div className="max-w-2xl space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-base font-semibold text-ink mb-1">Automation Rules</h2>
                <p className="text-xs text-ink-4">Trigger-based workflows that run automatically</p>
              </div>
              <button className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700">
                + New Rule
              </button>
            </div>

            <div className="space-y-3">
              {AUTOMATIONS.map(auto => (
                <div key={auto.id} className="bg-raised rounded-xl border border-line p-4">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <p className="text-xs font-medium text-ink">{auto.name}</p>
                        {auto.active
                          ? <span className="px-1.5 py-0.5 rounded-full text-2xs font-medium bg-green-500/15 text-success">active</span>
                          : <span className="px-1.5 py-0.5 rounded-full text-2xs font-medium dark:bg-gray-500/15 bg-gray-100 text-ink-3">inactive</span>
                        }
                      </div>
                      <div className="flex items-center gap-2 text-2xs">
                        <span className="text-ink-4">When:</span>
                        <span className="bg-hover text-ink-2 px-2 py-0.5 rounded">{auto.trigger}</span>
                        <ChevronRight size={10} className="text-ink-4" />
                        <span className="text-ink-4">Then:</span>
                        <span className="dark:bg-blue-500/10 bg-blue-50 text-accent px-2 py-0.5 rounded">{auto.action}</span>
                      </div>
                    </div>
                    <label className="relative flex cursor-pointer items-center shrink-0">
                      <input type="checkbox" defaultChecked={auto.active} className="sr-only peer" />
                      <div className="w-8 h-4 dark:bg-night-700 bg-gray-300 rounded-full peer peer-checked:bg-blue-600 transition-colors" />
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
              <h2 className="text-base font-semibold text-ink mb-1">Security</h2>
              <p className="text-xs text-ink-4">Authentication and access control</p>
            </div>
            <div className="dark:bg-amber-500/5 bg-amber-50 border dark:border-amber-500/20 border-amber-200 rounded-xl p-4 flex items-start gap-3">
              <AlertTriangle size={15} className="text-warning mt-0.5 shrink-0" />
              <div>
                <p className="text-xs font-medium text-warning">2FA not enabled for all users</p>
                <p className="text-2xs text-amber-400/70 mt-0.5">3 users have not enabled two-factor authentication.</p>
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
                <label key={item.label} className="flex items-center gap-3 cursor-pointer bg-raised border border-line rounded-xl p-3.5">
                  <input type="checkbox" defaultChecked={item.checked} className="rounded accent-blue-600 w-3.5 h-3.5 shrink-0" />
                  <span className="text-xs text-ink-2">{item.label}</span>
                  {item.checked && <Check size={12} className="text-success ml-auto" />}
                </label>
              ))}
            </div>
          </div>
        )}

        {activeSection === 'notifications' && (
          <div className="max-w-2xl space-y-6">
            <div>
              <h2 className="text-base font-semibold text-ink mb-1">Notifications</h2>
              <p className="text-xs text-ink-4">Configure alerts across channels</p>
            </div>
            {[
              { channel: 'Email', events: ['Critical stock alert', 'Expiry alert', 'Integration failure', 'New order'] },
              { channel: 'Webhook', events: ['Order status change', 'Stock adjustment', 'Sync complete'] },
              { channel: 'Slack', events: ['Critical stock alert', 'Integration failure'] },
            ].map(section => (
              <div key={section.channel} className="bg-raised rounded-xl border border-line overflow-hidden">
                <div className="px-4 py-3 border-b border-line flex items-center justify-between">
                  <p className="text-xs font-semibold text-ink">{section.channel}</p>
                  <span className="text-2xs text-success bg-green-500/15 px-2 py-0.5 rounded-full">configured</span>
                </div>
                <div className="p-4 space-y-2">
                  {section.events.map(ev => (
                    <label key={ev} className="flex items-center gap-3 cursor-pointer">
                      <input type="checkbox" defaultChecked className="rounded accent-blue-600 w-3 h-3" />
                      <span className="text-xs text-ink-2">{ev}</span>
                    </label>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}

        {!['company', 'integrations', 'inventory', 'localization', 'automation', 'security', 'notifications', 'tax', 'pricing'].includes(activeSection) && (
          <div className="max-w-2xl">
            <h2 className="text-base font-semibold text-ink mb-1 capitalize">{activeSection}</h2>
            <p className="text-xs text-ink-4">
              {SECTIONS.find(s => s.id === activeSection)?.description}
            </p>
            <div className="mt-8 flex flex-col items-center text-center">
              <div className="w-12 h-12 bg-raised border border-line rounded-xl flex items-center justify-center mb-3">
                {(() => { const Icon = SECTIONS.find(s => s.id === activeSection)?.icon ?? Zap; return <Icon size={20} className="text-ink-4" /> })()}
              </div>
              <p className="text-xs text-ink-4">Configuration coming soon</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

// ── Integrations settings section ─────────────────────────────────────────────

const STATUS_STYLES: Record<Connection['status'], string> = {
  connected: 'bg-green-500/15 text-success',
  error: 'bg-red-500/15 text-danger',
}

function IntegrationsSettingsSection({ onConfigure }: { onConfigure: (connection: Connection) => void }) {
  const { token } = useAuth()
  const { connections, loading, error } = useConnections(token ?? '')

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-base font-semibold text-ink mb-1">Integrations</h2>
        <p className="text-xs text-ink-4">Connected external services. Configure sync schedules for each.</p>
      </div>

      <AIIntegrationCard />

      <ApiKeysCard />

      <RecommendationsSettingsCard />

      {loading && <p className="text-xs text-ink-4">Loading…</p>}
      {error && <p className="text-xs text-danger">{error}</p>}

      {!loading && connections.length === 0 && (
        <div className="bg-raised rounded-xl border border-line p-6 text-center">
          <p className="text-xs text-ink-4">No integrations connected yet.</p>
        </div>
      )}

      <div className="space-y-3">
        {connections.map(conn => (
          <div key={conn.id} className="bg-raised rounded-xl border border-line p-4 flex items-center justify-between gap-4">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <p className="text-xs font-medium text-ink capitalize">{conn.provider}</p>
                <span className={clsx('px-1.5 py-0.5 rounded-full text-2xs font-medium capitalize', STATUS_STYLES[conn.status])}>
                  {conn.status}
                </span>
              </div>
              <p className="text-2xs text-ink-4 truncate mt-0.5">{conn.store_url}</p>
            </div>
            {conn.provider === 'woocommerce' ? (
              <button
                onClick={() => onConfigure(conn)}
                className="px-3 py-1.5 rounded-lg bg-fill text-ink-2 text-xs font-medium hover:bg-hover transition-colors shrink-0"
              >
                Configure
              </button>
            ) : (
              <span className="text-2xs text-ink-4 shrink-0">Coming soon</span>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

// Tunes the FP-Growth cross-sell mining GetCartRecommendationsUseCase runs
// for every storefront (e.g. the WooCommerce plugin) request -- lives next
// to ApiKeysCard since both are about that same storefront-facing
// recommendations feature. All three are quality levers, not speed ones
// (Apriori and FP-Growth mine identical itemsets for the same minSupport):
// they decide what counts as a real cross-sell pattern versus coincidence,
// not how fast the mining runs.
function RecommendationsSettingsCard() {
  const { token } = useAuth()
  const { settings, save } = useTenantSettings(token ?? '')

  const [confidencePercent, setConfidencePercent] = useState('')
  const [minSupport, setMinSupport] = useState('')
  const [lookbackDays, setLookbackDays] = useState('')
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [saved, setSaved] = useState(false)

  if (!hydrated && settings) {
    setConfidencePercent(settings.bundle_confidence_threshold ? (settings.bundle_confidence_threshold * 100).toString() : '')
    setMinSupport(settings.recommendations_min_support ? settings.recommendations_min_support.toString() : '')
    setLookbackDays(settings.recommendations_lookback_days ? settings.recommendations_lookback_days.toString() : '')
    setHydrated(true)
  }

  async function handleSave() {
    setSaveError('')
    setSaved(false)
    const confidence = confidencePercent.trim() === '' ? 0 : parseFloat(confidencePercent) / 100
    const support = minSupport.trim() === '' ? 0 : Math.round(parseFloat(minSupport))
    const days = lookbackDays.trim() === '' ? 0 : Math.round(parseFloat(lookbackDays))
    if (confidence < 0 || confidence > 1) {
      setSaveError('Bundle confidence must be between 0 and 100%.')
      return
    }
    if (support < 0) {
      setSaveError('Minimum support cannot be negative.')
      return
    }
    if (days < 0) {
      setSaveError('Lookback window cannot be negative.')
      return
    }
    setSaving(true)
    try {
      await save({
        ...toInput(settings),
        bundle_confidence_threshold: confidence,
        recommendations_min_support: support,
        recommendations_lookback_days: days,
      })
      setSaved(true)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save recommendation settings')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <h3 className="text-sm font-semibold text-ink">Cross-sell recommendations</h3>
      <p className="text-2xs text-ink-4">
        Tunes the pattern-mining behind "You may also like" and "Frequently bought together" on connected storefronts. Leave blank to use CommerceOS' defaults.
      </p>

      <div className="grid grid-cols-3 gap-3">
        <div>
          <label htmlFor="bundle-confidence" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
            Bundle confidence (%)
          </label>
          <input
            id="bundle-confidence"
            type="number" min={0} max={100} step={1}
            value={confidencePercent}
            onChange={e => { setConfidencePercent(e.target.value); setSaved(false) }}
            placeholder="30"
            className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div>
          <label htmlFor="min-support" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
            Minimum support (orders)
          </label>
          <input
            id="min-support"
            type="number" min={0} step={1}
            value={minSupport}
            onChange={e => { setMinSupport(e.target.value); setSaved(false) }}
            placeholder="2"
            className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div>
          <label htmlFor="lookback-days" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
            Lookback window (days)
          </label>
          <input
            id="lookback-days"
            type="number" min={0} step={1}
            value={lookbackDays}
            onChange={e => { setLookbackDays(e.target.value); setSaved(false) }}
            placeholder="180"
            className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
      </div>

      <ul className="text-2xs text-ink-4 space-y-0.5 list-disc list-inside">
        <li><span className="font-medium">Bundle confidence</span> — of orders containing a cart item, the % that must also contain a candidate before it's flagged bundle-worthy. Higher = fewer, more reliable bundle suggestions.</li>
        <li><span className="font-medium">Minimum support</span> — orders a product pairing must appear in before it counts as a real pattern rather than coincidence. Raise this on a high-volume store to cut noise; keep it low on a small catalog or recommendations may disappear entirely.</li>
        <li><span className="font-medium">Lookback window</span> — how far back order history is mined. Shorter favors recent behavior over stale, since-discontinued patterns; longer gives more data for a seasonal or slow-moving catalog.</li>
      </ul>

      {saveError && <p className="text-xs text-danger">{saveError}</p>}
      {saved && <p className="text-xs text-success">Saved.</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
      >
        {saving ? 'Saving…' : 'Save recommendation settings'}
      </button>
    </div>
  )
}

// Tenant-wide, not tied to any one Connection -- a storefront (e.g. the
// WooCommerce plugin) calls CommerceOS' cart-recommendations API with one
// of these, and that call only needs a tenant, not a specific integration
// connection (a tenant can have several, see the connections list above).
// Several named, independently revocable keys, mirroring
// ProviderWebhooksPanel's list+create+delete shape.
function ApiKeysCard() {
  const { token } = useAuth()
  const [keys, setKeys] = useState<ApiKey[]>([])
  const [channels, setChannels] = useState<SalesChannel[]>([])
  const [loading, setLoading] = useState(true)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [salesChannelId, setSalesChannelId] = useState('')
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')
  const [justCreatedKey, setJustCreatedKey] = useState('')
  const [copied, setCopied] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)

  const internalChannels = channels.filter(c => c.type === 'internal')

  useEffect(() => {
    let cancelled = false
    Promise.all([settingsApi.listApiKeys(token ?? ''), integrationsApi.listSalesChannels(token ?? '')])
      .then(([keysResult, channelsResult]) => { if (!cancelled) { setKeys(keysResult); setChannels(channelsResult) } })
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load API keys') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setCreating(true)
    try {
      const created = await settingsApi.createApiKey(token ?? '', name.trim(), description.trim(), salesChannelId || null)
      setKeys(prev => [created, ...prev])
      setJustCreatedKey(created.key)
      setName('')
      setDescription('')
      setSalesChannelId('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create API key')
    } finally {
      setCreating(false)
    }
  }

  function channelName(id: string | null): string {
    if (!id) return 'All channels'
    return channels.find(c => c.id === id)?.name ?? 'Unknown channel'
  }

  async function handleRevoke(id: string) {
    setBusyId(id)
    setError('')
    try {
      await settingsApi.revokeApiKey(token ?? '', id)
      setKeys(prev => prev.map(k => (k.id === id ? { ...k, revoked_at: new Date().toISOString() } : k)))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to revoke API key')
    } finally {
      setBusyId(null)
    }
  }

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(justCreatedKey)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard access can be denied (permissions, insecure context) --
      // the value is already shown selectable on screen, so this is a
      // convenience failure, not a blocker; no error surfaced for it.
    }
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <div>
        <p className="text-xs font-medium text-ink">API keys</p>
        <p className="text-2xs text-ink-4 mt-0.5">
          Each key lets a storefront (e.g. the CommerceOS Sync WordPress plugin) call CommerceOS' cart recommendations
          API. Name each key for the storefront it's for, so you can revoke just that one later without affecting others.
        </p>
      </div>

      {justCreatedKey && (
        <div>
          <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
            New key (shown once -- copy it now)
          </label>
          <div className="flex items-center gap-2">
            <code className="flex-1 px-3 py-2 text-2xs font-mono rounded-lg bg-sunken border border-line text-ink-2 overflow-x-auto whitespace-nowrap">
              {justCreatedKey}
            </code>
            <button
              type="button"
              onClick={handleCopy}
              className="px-2.5 py-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors shrink-0"
            >
              {copied ? 'Copied' : 'Copy'}
            </button>
          </div>
          <p className="mt-1.5 text-2xs text-warning">
            This won't be shown again -- if you lose it, revoke it below and create a new one.
          </p>
        </div>
      )}

      {error && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error}</p>}

      <form onSubmit={handleCreate} className="flex items-end gap-2">
        <div className="flex-1">
          <label htmlFor="api-key-name" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
            Name
          </label>
          <input
            id="api-key-name" required value={name} onChange={e => setName(e.target.value)}
            placeholder="e.g. Main storefront"
            className="w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink"
          />
        </div>
        <div className="flex-1">
          <label htmlFor="api-key-description" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
            Description
          </label>
          <input
            id="api-key-description" value={description} onChange={e => setDescription(e.target.value)}
            placeholder="Optional"
            className="w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink"
          />
        </div>
        {internalChannels.length > 0 && (
          <div className="flex-1">
            <label htmlFor="api-key-channel" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              Sales channel
            </label>
            <select
              id="api-key-channel" value={salesChannelId} onChange={e => setSalesChannelId(e.target.value)}
              className="w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink"
            >
              <option value="">All channels (tenant-wide)</option>
              {internalChannels.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
            </select>
          </div>
        )}
        <button
          type="submit" disabled={creating}
          className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
        >
          {creating ? 'Creating…' : 'Create key'}
        </button>
      </form>

      {loading ? (
        <p className="text-xs text-ink-4">Loading…</p>
      ) : keys.length === 0 ? (
        <p className="text-xs text-ink-4">No keys yet -- create one above.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-left text-ink-4 border-b border-line">
                <th className="py-2 pr-3 font-medium">Name</th>
                <th className="py-2 pr-3 font-medium">Description</th>
                <th className="py-2 pr-3 font-medium">Scope</th>
                <th className="py-2 pr-3 font-medium">Status</th>
                <th className="py-2 pr-3 font-medium"></th>
              </tr>
            </thead>
            <tbody>
              {keys.map(k => (
                <tr key={k.id} className="border-b border-line last:border-0">
                  <td className="py-2 pr-3 text-ink-2">{k.name}</td>
                  <td className="py-2 pr-3 text-ink-3 truncate max-w-[220px]">{k.description}</td>
                  <td className="py-2 pr-3 text-ink-3">{channelName(k.sales_channel_id)}</td>
                  <td className="py-2 pr-3">
                    {k.revoked_at ? (
                      <span className="text-2xs text-ink-4">revoked</span>
                    ) : (
                      <span className="text-2xs text-success bg-green-500/15 px-2 py-0.5 rounded-full">active</span>
                    )}
                  </td>
                  <td className="py-2 pr-3">
                    {!k.revoked_at && (
                      <button
                        onClick={() => handleRevoke(k.id)}
                        disabled={busyId === k.id}
                        className="px-2 py-1 rounded bg-fill text-ink-3 hover:text-danger disabled:opacity-50 transition-colors"
                      >
                        Revoke
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
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
  const [fallbackModels, setFallbackModels] = useState('')
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  const openConfigure = () => {
    setProvider(connection?.provider ?? 'openai')
    setApiKey('')
    setBaseUrl(connection?.base_url ?? '')
    setModel(connection?.model ?? '')
    setFallbackModels(connection?.fallback_models ?? '')
    setFormError('')
    setConfiguring(true)
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setFormError('')
    try {
      await connect({ provider, apiKey, baseUrl, model, fallbackModels })
      setConfiguring(false)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to connect')
    } finally {
      setSaving(false)
    }
  }

  const handleDisconnect = async () => {
    if (!window.confirm('Disconnect AI provider? Procurement PDF document parsing and WhatsApp AI-assisted parsing will stop working until reconfigured.')) return
    await disconnect()
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0 flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-cyan-600 flex items-center justify-center text-white shrink-0">
            <Bot size={14} />
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <p className="text-xs font-medium text-ink">AI</p>
              {connection && (
                <span className={clsx('px-1.5 py-0.5 rounded-full text-2xs font-medium capitalize', STATUS_STYLES[connection.status])}>
                  {connection.status}
                </span>
              )}
            </div>
            <p className="text-2xs text-ink-4 mt-0.5">
              {loading ? 'Loading…' : connection
                ? AI_PROVIDER_LABELS[connection.provider]
                : 'Used for procurement document parsing (PDF invoices) and optional WhatsApp AI-assisted message parsing. Not configured.'}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <button
            onClick={openConfigure}
            className="px-3 py-1.5 rounded-lg bg-fill text-ink-2 text-xs font-medium hover:bg-hover transition-colors"
          >
            {connection ? 'Configure' : 'Connect'}
          </button>
          {connection && (
            <button
              onClick={handleDisconnect}
              className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-3 hover:text-danger transition-colors"
            >
              Disconnect
            </button>
          )}
        </div>
      </div>
      {error && <p className="mt-2 text-2xs text-danger">{error}</p>}

      {configuring && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={() => setConfiguring(false)} />
          <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-ink">Configure AI</h3>
              <button onClick={() => setConfiguring(false)} aria-label="Close">
                <X size={14} className="text-ink-4" />
              </button>
            </div>
            <form onSubmit={handleSave} className="space-y-3">
              <div>
                <label htmlFor="ai-provider" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  Provider
                </label>
                <select
                  id="ai-provider"
                  value={provider}
                  onChange={e => setProvider(e.target.value as AIProvider)}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                >
                  {(Object.keys(AI_PROVIDER_LABELS) as AIProvider[]).map(p => (
                    <option key={p} value={p}>{AI_PROVIDER_LABELS[p]}</option>
                  ))}
                </select>
              </div>
              {provider !== 'ollama' && (
                <div>
                  <label htmlFor="ai-api-key" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
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
                    className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
              )}
              <div>
                <label htmlFor="ai-base-url" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
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
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div>
                <label htmlFor="ai-model" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  Model (optional)
                </label>
                <input
                  id="ai-model"
                  type="text"
                  value={model}
                  onChange={e => setModel(e.target.value)}
                  placeholder="Uses the provider's default"
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              {provider === 'gemini' && (
                <div>
                  <label htmlFor="ai-fallback-models" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Fallback models (optional)
                  </label>
                  <input
                    id="ai-fallback-models"
                    type="text"
                    value={fallbackModels}
                    onChange={e => setFallbackModels(e.target.value)}
                    placeholder="gemini-3.5-flash-lite, gemini-3.1-flash-lite"
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  <p className="mt-1.5 text-2xs text-ink-4">
                    Comma-separated. Tried in order when Model hits a rate limit (Gemini's free tier caps some models at 20 requests/day, others at 500).
                  </p>
                </div>
              )}
              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
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
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <h3 className="text-sm font-semibold text-ink">Sync & expiry rules</h3>
      <div className="flex items-center justify-between">
        <div>
          <span className="text-xs text-ink-2 block">Enable inventory sync rules</span>
          <span className="text-2xs text-ink-4">
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
          <div className="w-8 h-4 dark:bg-night-700 bg-gray-300 rounded-full peer peer-checked:bg-blue-600 transition-colors" />
          <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
        </label>
      </div>
      <div className="max-w-[220px]">
        <label htmlFor="batch-expiry-threshold-days" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
          Auto-expire batches after (days)
        </label>
        <input
          id="batch-expiry-threshold-days"
          type="number"
          min={0}
          value={thresholdDays}
          onChange={e => { setThresholdDays(parseInt(e.target.value, 10) || 0); setSaved(false) }}
          className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
        />
      </div>
      {saveError && <p className="text-xs text-danger">{saveError}</p>}
      {saved && <p className="text-xs text-success">Saved.</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
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
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <h3 className="text-sm font-semibold text-ink">Pricing & markup</h3>
      <div className="max-w-[220px]">
        <label htmlFor="default-target-markup" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
          Default target markup (%)
        </label>
        <input
          id="default-target-markup"
          type="number" min={0} step={0.01}
          value={markupPercent}
          onChange={e => { setMarkupPercent(e.target.value); setSaved(false) }}
          placeholder="e.g. 30"
          className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
        />
        <p className="text-2xs text-ink-4 mt-1">
          Markup is profit as a % of cost (e.g. 300% markup means selling at 4x cost) -- no upper bound, unlike margin.
          Used to compute each product's recommended price from its cost price, unless its category has its own target markup set.
        </p>
      </div>
      {saveError && <p className="text-xs text-danger">{saveError}</p>}
      {saved && <p className="text-xs text-success">Saved.</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
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
        <h2 className="text-base font-semibold text-ink mb-1">Localization</h2>
        <p className="text-xs text-ink-4">Currency, timezone, date formats</p>
      </div>
      {loading && <p className="text-xs text-ink-4">Loading…</p>}
      <div className="space-y-4">
        {fields.map(({ id, label, value, onChange, options }) => (
          <div key={id}>
            <label htmlFor={id} className="block text-xs font-medium text-ink-2 mb-1.5">{label}</label>
            <select
              id={id}
              value={value}
              onChange={e => { onChange(e.target.value); setSaved(false) }}
              className="w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-blue-500"
            >
              {options.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
            </select>
          </div>
        ))}
      </div>
      {(error || saveError) && <p className="text-xs text-danger">{error || saveError}</p>}
      {saved && <p className="text-xs text-success">Saved.</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50"
      >
        {saving ? 'Saving…' : 'Save Changes'}
      </button>

      <ExchangeRatesCard />
    </div>
  )
}

// ExchangeRatesCard manages the tenant's fx conversion rates -- the
// fallback a SalesChannel running in a non-default currency uses to price
// a SKU/coupon with no explicit per-currency override (see domain/fx). A
// rate converts 1 minor-unit of the tenant's default currency into that
// many minor-units of the target currency.
function ExchangeRatesCard() {
  const { token } = useAuth()
  const { settings } = useTenantSettings(token ?? '')
  const defaultCurrencyLabel = settings?.default_currency || 'your default currency'
  const { rates, loading, error, upsert, remove } = useExchangeRates(token ?? '')
  const [currencyCode, setCurrencyCode] = useState('')
  const [rateValue, setRateValue] = useState('')
  const [formError, setFormError] = useState('')
  const [saving, setSaving] = useState(false)

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    const parsed = parseFloat(rateValue)
    if (!parsed || parsed <= 0) {
      setFormError('Rate must be a positive number')
      return
    }
    setSaving(true)
    try {
      await upsert({ currency_code: currencyCode.toUpperCase(), rate_basis_points: Math.round(parsed * 10000) })
      setCurrencyCode('')
      setRateValue('')
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save exchange rate')
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(code: string) {
    if (!window.confirm(`Delete the exchange rate for ${code}? Any channel priced in it via conversion (not a manual override) will stop pricing until a new rate is set.`)) return
    setFormError('')
    try {
      await remove(code)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to delete exchange rate')
    }
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <div>
        <h3 className="text-sm font-semibold text-ink">Exchange rates</h3>
        <p className="text-xs text-ink-4 mt-0.5">
          How much 1 unit of your default currency is worth in another currency -- used to price a sales channel
          running in that currency, unless a product/coupon has a manual override.
        </p>
      </div>

      {error && <p className="text-xs text-danger">{error}</p>}
      {loading && <p className="text-xs text-ink-4">Loading…</p>}

      <div className="space-y-1.5">
        {rates.map(r => (
          <div key={r.currency_code} className="flex items-center justify-between text-xs text-ink-2 py-1">
            <span>1 {defaultCurrencyLabel} = <span className="font-medium">{(r.rate_basis_points / 10000).toFixed(4)}</span> {r.currency_code}</span>
            <button
              onClick={() => handleDelete(r.currency_code)}
              className="text-2xs text-ink-4 hover:text-danger transition-colors"
            >
              Delete
            </button>
          </div>
        ))}
        {!loading && rates.length === 0 && <p className="text-xs text-ink-4">No exchange rates configured yet.</p>}
      </div>

      <form onSubmit={handleAdd} className="flex items-end gap-2 pt-2 border-t border-line">
        <div className="w-20">
          <label htmlFor="fx-currency" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">Currency</label>
          <input
            id="fx-currency"
            value={currencyCode}
            maxLength={3}
            placeholder="USD"
            onChange={e => setCurrencyCode(e.target.value.toUpperCase())}
            className="w-full px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div className="w-28">
          <label htmlFor="fx-rate" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">Rate</label>
          <input
            id="fx-rate"
            value={rateValue}
            placeholder="1.0876"
            onChange={e => setRateValue(e.target.value)}
            className="w-full px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <button
          type="submit"
          disabled={saving || currencyCode.length !== 3 || !rateValue}
          className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
        >
          {saving ? 'Saving…' : 'Add / update'}
        </button>
      </form>
      {formError && <p className="text-xs text-danger">{formError}</p>}
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

  // Add/edit tax class form open state is a URL (/settings/tax/classes/new
  // or /<id>), so it survives a refresh; the form fills once the class list
  // has it.
  const classEditor = useRoutedEditor(TAX_CLASSES_PATH)
  const creatingClass = classEditor.editingId === NEW_RECORD
  const editingClass = classEditor.editingId && !creatingClass
    ? classes.find(c => c.id === classEditor.editingId) ?? null
    : null
  const [classForm, setClassForm] = useState<TaxClassFormState>({ name: '', isDefault: false })
  const [classSaving, setClassSaving] = useState(false)
  const [classFormError, setClassFormError] = useState('')
  const [rowError, setRowError] = useState('')

  const classFormReady = useInitOnce(classEditor.editingId, creatingClass || !!editingClass, () => {
    setClassForm(editingClass ? { name: editingClass.name, isDefault: editingClass.is_default } : { name: '', isDefault: false })
    setClassFormError('')
  })
  const openCreateClass = classEditor.openNew
  const openEditClass = (c: TaxClass) => classEditor.openEdit(c.id)
  const closeClassForm = classEditor.close

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
        <h2 className="text-base font-semibold text-ink mb-1">Tax</h2>
        <p className="text-xs text-ink-4">Tax classes and their per-country rates</p>
      </div>

      <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
        <h3 className="text-sm font-semibold text-ink">Store defaults</h3>
        <div className="flex items-center justify-between">
          <span className="text-xs text-ink-2">Prices include tax</span>
          <label className="relative flex items-center shrink-0 cursor-pointer">
            <input
              type="checkbox"
              role="switch"
              aria-label="Prices include tax"
              checked={pricesIncludeTax}
              onChange={e => { setPricesIncludeTax(e.target.checked); setDefaultsSaved(false) }}
              className="sr-only peer"
            />
            <div className="w-8 h-4 dark:bg-night-700 bg-gray-300 rounded-full peer peer-checked:bg-blue-600 transition-colors" />
            <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
          </label>
        </div>
        <div className="max-w-[160px]">
          <label htmlFor="default-tax-country" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
            Default tax country
          </label>
          <input
            id="default-tax-country"
            value={defaultTaxCountry}
            maxLength={2}
            onChange={e => { setDefaultTaxCountry(e.target.value.toUpperCase()); setDefaultsSaved(false) }}
            className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        {defaultsError && <p className="text-xs text-danger">{defaultsError}</p>}
        {defaultsSaved && <p className="text-xs text-success">Saved.</p>}
        <button
          onClick={handleSaveDefaults}
          disabled={savingDefaults}
          className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
        >
          {savingDefaults ? 'Saving…' : 'Save tax defaults'}
        </button>
      </div>

      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-ink">Tax classes</h3>
        <button
          onClick={openCreateClass}
          className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
        >
          Add tax class
        </button>
      </div>
      {classesError && <p className="text-xs text-danger">{classesError}</p>}
      {ratesError && <p className="text-xs text-danger">{ratesError}</p>}
      {rowError && <p className="text-xs text-danger">{rowError}</p>}
      {classesLoading && <p className="text-xs text-ink-4">Loading…</p>}

      <div className="space-y-3">
        {classes.map(c => (
          <div key={c.id} className="bg-raised rounded-xl border border-line p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-ink">{c.name}</span>
                {c.is_default && (
                  <span className="text-2xs px-2 py-0.5 rounded-full dark:bg-blue-500/10 bg-blue-50 text-accent">Default</span>
                )}
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => openEditClass(c)}
                  className="py-1 px-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                >
                  Edit
                </button>
                <button
                  onClick={() => handleDeleteClass(c)}
                  className="py-1 px-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-danger transition-colors"
                >
                  Delete
                </button>
              </div>
            </div>
            <div className="mt-3 space-y-1.5">
              {rates.filter(r => r.tax_class_id === c.id).map(r => (
                <div key={r.id} className="flex items-center justify-between text-xs text-ink-2">
                  <span>{r.country}</span>
                  <span>{(r.rate_basis_points / 100).toFixed(2)}%</span>
                  <button
                    onClick={() => handleRemoveRate(r.id)}
                    className="text-2xs text-ink-4 hover:text-danger transition-colors flex items-center gap-1"
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

      {classFormReady && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={closeClassForm} />
          <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-ink">
                {editingClass ? 'Edit tax class' : 'Add tax class'}
              </h3>
              <button onClick={closeClassForm} aria-label="Close">
                <X size={14} className="text-ink-4" />
              </button>
            </div>
            <form onSubmit={handleClassSubmit} className="space-y-3">
              <div>
                <label htmlFor="tax-class-name" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  Name
                </label>
                <input
                  id="tax-class-name"
                  type="text"
                  required
                  autoFocus
                  value={classForm.name}
                  onChange={e => setClassForm(f => ({ ...f, name: e.target.value }))}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div className="flex items-center justify-between">
                <span className="text-2xs text-ink-4 uppercase tracking-wider">Default tax class</span>
                <label className="relative flex items-center shrink-0 cursor-pointer">
                  <input
                    type="checkbox"
                    role="switch"
                    aria-label="Default tax class"
                    checked={classForm.isDefault}
                    onChange={e => setClassForm(f => ({ ...f, isDefault: e.target.checked }))}
                    className="sr-only peer"
                  />
                  <div className="w-8 h-4 dark:bg-night-700 bg-gray-300 rounded-full peer peer-checked:bg-blue-600 transition-colors" />
                  <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
                </label>
              </div>
              {classFormError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{classFormError}</p>
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
      className="mt-3 pt-3 border-t border-line flex items-end gap-2"
    >
      <div>
        <label htmlFor={`tax-rate-country-${taxClassId}`} className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
          Country
        </label>
        <input
          id={`tax-rate-country-${taxClassId}`}
          value={country}
          maxLength={2}
          onChange={e => setCountry(e.target.value.toUpperCase())}
          className="w-16 px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none"
        />
      </div>
      <div>
        <label htmlFor={`tax-rate-percent-${taxClassId}`} className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
          Rate %
        </label>
        <input
          id={`tax-rate-percent-${taxClassId}`}
          type="number"
          step="0.01"
          value={ratePercent}
          onChange={e => setRatePercent(e.target.value)}
          className="w-20 px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none"
        />
      </div>
      <button
        type="submit"
        className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
      >
        Add rate
      </button>
    </form>
  )
}
