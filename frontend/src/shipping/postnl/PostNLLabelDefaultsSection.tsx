import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { PACKAGE_TYPES, type ShippingConnectionSettings } from '../types'
import { labelDefaultsOf, type LabelDefaults } from './labelDefaults'
import { apiErrorText } from '../../api/apiErrorText'
import { SettingsSection, inputClass, labelClass, primaryButtonClass } from './settingsSectionUi'

interface PostNLLabelDefaultsSectionProps {
  settings: ShippingConnectionSettings | null
  // PACKAGE_TYPES ids the provider offers.
  packageTypeIds: readonly number[]
  // Rejects on failure so the section can show why.
  onSave: (defaults: LabelDefaults) => Promise<void>
}

type Toggle = 'default_signature_required' | 'default_only_recipient' | 'insure_by_default' | 'auto_print_enabled'

// Label format, default package type and the per-label options PostNL
// labels start with.
export function PostNLLabelDefaultsSection({ settings, packageTypeIds, onSave }: PostNLLabelDefaultsSectionProps) {
  const { t } = useTranslation(['shipping', 'common'])
  const [defaults, setDefaults] = useState(() => labelDefaultsOf(settings, packageTypeIds))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  function set<K extends keyof LabelDefaults>(key: K, value: LabelDefaults[K]) {
    setSaved(false)
    setDefaults(d => ({ ...d, [key]: value }))
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      await onSave(defaults)
      setSaved(true)
    } catch (err) {
      setError(apiErrorText(err) ?? t('myparcel.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  const toggle = (key: Toggle, label: string, hint?: string) => (
    <div>
      <label className="flex items-center gap-2 text-xs text-ink-2">
        <input type="checkbox" checked={defaults[key]} onChange={e => set(key, e.target.checked)} />
        {label}
      </label>
      {hint && <p className="text-2xs text-ink-4 mt-1">{hint}</p>}
    </div>
  )

  return (
    <SettingsSection id="postnl-label-defaults" title={t('postnlSettings.labelDefaults')} body={t('postnlSettings.labelDefaultsBody')}>
      <form onSubmit={handleSubmit} className="space-y-3">
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="postnl-label-format" className={labelClass}>{t('myparcel.labelFormat')}</label>
            <select id="postnl-label-format" value={defaults.default_label_format}
              onChange={e => set('default_label_format', e.target.value === 'zpl' ? 'zpl' : 'pdf')} className={inputClass}>
              <option value="pdf">PDF</option>
              <option value="zpl">{t('myparcel.zpl')}</option>
            </select>
          </div>
          <div>
            <label htmlFor="postnl-package-type" className={labelClass}>{t('myparcel.defaultPackageType')}</label>
            <select id="postnl-package-type" value={defaults.default_package_type}
              onChange={e => set('default_package_type', Number(e.target.value))} className={inputClass}>
              {PACKAGE_TYPES.filter(p => packageTypeIds.includes(p.id)).map(p => <option key={p.id} value={p.id}>{p.label}</option>)}
            </select>
          </div>
        </div>
        {toggle('default_signature_required', t('myparcel.signatureRequired'))}
        {toggle('default_only_recipient', t('myparcel.onlyRecipient'))}
        {toggle('insure_by_default', t('myparcel.insureDefault'), t('postnlSettings.insureHint'))}
        {toggle('auto_print_enabled', t('myparcel.autoPrint'), t('myparcel.autoPrintHint'))}
        {error && <p role="alert" className="text-xs text-danger">{error}</p>}
        {saved && <p role="status" className="text-xs text-success">{t('postnlSettings.defaultsSaved')}</p>}
        <button type="submit" disabled={saving} className={primaryButtonClass}>
          {saving ? t('common:actions.saving') : t('postnlSettings.saveDefaults')}
        </button>
      </form>
    </SettingsSection>
  )
}
