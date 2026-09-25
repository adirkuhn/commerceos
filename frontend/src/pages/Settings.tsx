
import { Building2, Package, Truck, Globe, Zap, Shield, Bell, Plug, Percent, TrendingUp, PackageCheck } from 'lucide-react'
import clsx from 'clsx'
import { useLocation, useNavigate } from 'react-router-dom'
import { subPathSegments } from '../routes'
import { useStackNavigation } from '../routing/useStackNavigation'
import { ConnectionRoute } from '../integrations/ConnectionRoute'
import WooCommerceSyncSettingsPage from '../integrations/WooCommerceSyncSettingsPage'
import { PickingModeSection } from '../settings/PickingModeSection'
import { CompanyProfileSection } from '../settings/company/CompanyProfileSection'
import { NotificationPreferencesSection } from '../settings/notifications/NotificationPreferencesSection'
import { AutomationSection, SecuritySection } from '../settings/ManagedElsewhereSections'
import { lm2Origin } from '../auth/loginConfig'
import { SETTINGS_PATH } from '../settings/settingsPaths'
import { IntegrationsSettingsSection } from '../settings/IntegrationsSettingsSection'
import { InventorySyncRulesSection } from '../settings/InventorySyncRulesSection'
import { PricingMarginSection } from '../settings/PricingMarginSection'
import { LocalizationSettingsSection } from '../settings/LocalizationSettingsSection'
import { TaxSettingsSection } from '../settings/TaxSettingsSection'

interface SettingSection {
  id: string
  label: string
  icon: React.ElementType
  description: string
}

const SECTIONS: SettingSection[] = [
  { id: 'company', label: 'Company', icon: Building2, description: 'Business name, address, tax IDs' },
  { id: 'integrations', label: 'Integrations', icon: Plug, description: 'Connect and configure external services' },
  { id: 'inventory', label: 'Inventory rules', icon: Package, description: 'FEFO, min stock, reorder points' },
  { id: 'fulfillment', label: 'Fulfillment', icon: PackageCheck, description: 'Picking mode' },
  { id: 'shipping', label: 'Shipping rules', icon: Truck, description: 'Default carriers, label format, zones' },
  { id: 'localization', label: 'Localization', icon: Globe, description: 'Currency, timezone, language' },
  { id: 'tax', label: 'Tax', icon: Percent, description: 'Tax classes and rates' },
  { id: 'pricing', label: 'Pricing', icon: TrendingUp, description: 'Default target markup for recommended prices' },
  { id: 'automation', label: 'Automation', icon: Zap, description: 'Trigger-based rules and workflows' },
  { id: 'security', label: 'Security', icon: Shield, description: 'SSO, 2FA, IP restrictions' },
  { id: 'notifications', label: 'Notifications', icon: Bell, description: 'What shows in the bell and what is emailed' },
]

type SectionId = typeof SECTIONS[number]['id']
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
                aria-current={active ? 'page' : undefined}
                className={clsx(
                  'w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left mb-0.5 transition-colors',
                  active
                    ? 'bg-accent/10 text-accent'
                    : 'text-ink-3 hover:bg-raised',
                )}
              >
                <Icon size={14} className={active ? 'text-accent' : ''} />
                <span className="text-xs font-medium flex-1">{s.label}</span>
              </button>
            )
          })}
        </nav>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-6">
        {activeSection === 'company' && <CompanyProfileSection />}

        {activeSection === 'integrations' && <IntegrationsSettingsSection onConfigure={c => open(`${INTEGRATIONS_SECTION_PATH}/${encodeURIComponent(c.id)}`)} />}

        {activeSection === 'inventory' && (
          <div className="max-w-2xl space-y-6">
            <div>
              <h2 className="text-base font-semibold text-ink mb-1">Inventory rules</h2>
              <p className="text-xs text-ink-4">Configure stock management behavior</p>
            </div>

            <InventorySyncRulesSection />

            <div className="bg-raised rounded-xl border border-line overflow-hidden">
              <div className="px-4 py-3 border-b border-line">
                <h3 className="text-xs font-semibold text-ink">FEFO settings</h3>
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
                    <input type="checkbox" defaultChecked={defaultChecked as boolean} className="rounded accent-accent-solid w-3.5 h-3.5" />
                    <span className="text-xs text-ink-2">{label as string}</span>
                  </label>
                ))}
              </div>
            </div>

            <div className="bg-raised rounded-xl border border-line overflow-hidden">
              <div className="px-4 py-3 border-b border-line">
                <h3 className="text-xs font-semibold text-ink">Stock thresholds</h3>
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
                      className="w-24 px-2 py-1.5 text-xs rounded-lg bg-sunken border border-line text-ink outline-none focus:ring-1 focus:ring-focus text-right"
                    />
                  </div>
                ))}
              </div>
            </div>
            <button className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover">Save changes</button>
          </div>
        )}

        {activeSection === 'fulfillment' && <PickingModeSection />}
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

        {activeSection === 'automation' && <AutomationSection />}

        {activeSection === 'security' && <SecuritySection lm2Url={lm2Origin()} />}

        {activeSection === 'notifications' && <NotificationPreferencesSection />}

      </div>
    </div>
  )
}
