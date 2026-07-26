import { useState } from 'react'
import {
  Building2, Package, Truck, Globe, Zap, Shield, Bell, CreditCard,
  ChevronRight, Check, AlertTriangle,
} from 'lucide-react'
import clsx from 'clsx'

interface SettingSection {
  id: string
  label: string
  icon: React.ElementType
  description: string
}

const SECTIONS: SettingSection[] = [
  { id: 'company', label: 'Company', icon: Building2, description: 'Business name, address, tax IDs' },
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

        {activeSection === 'localization' && (
          <div className="max-w-2xl space-y-6">
            <div>
              <h2 className="text-base font-semibold dark:text-gray-100 text-gray-900 mb-1">Localization</h2>
              <p className="text-xs dark:text-gray-500 text-gray-400">Currency, timezone, date formats</p>
            </div>
            <div className="space-y-4">
              {[
                { label: 'Default Currency', options: ['EUR — Euro', 'USD — US Dollar', 'GBP — British Pound', 'CHF — Swiss Franc'], def: 0 },
                { label: 'Timezone', options: ['Europe/Amsterdam (UTC+2)', 'Europe/London (UTC+1)', 'Europe/Paris (UTC+2)', 'America/New_York (UTC-4)'], def: 0 },
                { label: 'Date Format', options: ['DD/MM/YYYY', 'MM/DD/YYYY', 'YYYY-MM-DD'], def: 0 },
                { label: 'Language', options: ['English (EN)', 'Dutch (NL)', 'German (DE)', 'French (FR)'], def: 0 },
              ].map(({ label, options, def }) => (
                <div key={label}>
                  <label className="block text-xs font-medium dark:text-gray-300 text-gray-700 mb-1.5">{label}</label>
                  <select
                    defaultValue={options[def]}
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 outline-none focus:ring-1 focus:ring-blue-500"
                  >
                    {options.map(o => <option key={o}>{o}</option>)}
                  </select>
                </div>
              ))}
            </div>
            <button className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700">Save Changes</button>
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

        {!['company', 'inventory', 'localization', 'automation', 'security', 'notifications'].includes(activeSection) && (
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
