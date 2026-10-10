
import { Building2, Package, Truck, Globe, Zap, Shield, Bell, Plug, Percent, TrendingUp, PackageCheck } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
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
import { ExpiryAlertSection } from '../settings/ExpiryAlertSection'
import { PricingMarginSection } from '../settings/PricingMarginSection'
import { LocalizationSettingsSection } from '../settings/LocalizationSettingsSection'
import { TaxSettingsSection } from '../settings/TaxSettingsSection'

interface SettingSection {
  id: string
  label: string
  icon: React.ElementType
  description: string
}

const SECTION_IDS = ['company', 'integrations', 'inventory', 'fulfillment', 'shipping', 'localization', 'tax', 'pricing', 'automation', 'security', 'notifications'] as const

function sections(t: TFunction<['settings', 'common']>): SettingSection[] {
  return [
    { id: 'company', label: t('sections.company'), icon: Building2, description: t('sections.companyText') },
    { id: 'integrations', label: t('sections.integrations'), icon: Plug, description: t('sections.integrationsText') },
    { id: 'inventory', label: t('sections.inventory'), icon: Package, description: t('sections.inventoryText') },
    { id: 'fulfillment', label: t('sections.fulfillment'), icon: PackageCheck, description: t('sections.fulfillmentText') },
    { id: 'shipping', label: t('sections.shipping'), icon: Truck, description: t('sections.shippingText') },
    { id: 'localization', label: t('sections.localization'), icon: Globe, description: t('sections.localizationText') },
    { id: 'tax', label: t('sections.tax'), icon: Percent, description: t('sections.taxText') },
    { id: 'pricing', label: t('sections.pricing'), icon: TrendingUp, description: t('sections.pricingText') },
    { id: 'automation', label: t('sections.automation'), icon: Zap, description: t('sections.automationText') },
    { id: 'security', label: t('sections.security'), icon: Shield, description: t('sections.securityText') },
    { id: 'notifications', label: t('sections.notifications'), icon: Bell, description: t('sections.notificationsText') },
  ]
}

type SectionId = typeof SECTION_IDS[number]
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
  const { t } = useTranslation(['settings', 'common'])
  const { open, exitTo } = useStackNavigation()
  const [sectionSegment, connectionId] = subPathSegments(location.pathname, SETTINGS_PATH)
  const activeSection: SectionId = (SECTION_IDS as readonly string[]).includes(sectionSegment) ? sectionSegment as SectionId : DEFAULT_SECTION
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
          <p className="text-xs font-semibold text-ink">{t('title')}</p>
        </div>
        <nav className="flex-1 py-2 px-2">
          {sections(t).map(s => {
            const Icon = s.icon
            const active = activeSection === s.id
            return (
              <button
                key={s.id}
                onClick={() => setActiveSection(s.id as SectionId)}
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
              <h2 className="text-base font-semibold text-ink mb-1">{t('sections.inventory')}</h2>
              <p className="text-xs text-ink-4">{t('inventory.subtitle')}</p>
            </div>

            <InventorySyncRulesSection />
            <ExpiryAlertSection />

            <div className="bg-raised rounded-xl border border-line overflow-hidden">
              <div className="px-4 py-3 border-b border-line">
                <h3 className="text-xs font-semibold text-ink">{t('inventory.fefoTitle')}</h3>
                <p className="text-2xs text-ink-4 mt-0.5">{t('inventory.fefoText')}</p>
              </div>
              <div className="p-4 space-y-3">
                {[
                  [t('inventory.fefoEnable'), true],
                  [t('inventory.fefoAlert'), true],
                  [t('inventory.fefoFlag'), true],
                  [t('inventory.fefoBlock'), false],
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
                <h3 className="text-xs font-semibold text-ink">{t('inventory.thresholdsTitle')}</h3>
              </div>
              <div className="p-4 space-y-3">
                {[
                  [t('inventory.minStock'), '20'],
                  [t('inventory.critical'), '15'],
                  [t('inventory.overstock'), '200'],
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
            <button className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover">{t('common:actions.saveChanges')}</button>
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
              <h2 className="text-base font-semibold text-ink mb-1">{t('sections.pricing')}</h2>
              <p className="text-xs text-ink-4">{t('pricing.subtitle')}</p>
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
