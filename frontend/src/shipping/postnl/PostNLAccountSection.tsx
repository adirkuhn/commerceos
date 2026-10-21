import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import type { PostNLAccount, UpdatePostNLAccountInput } from '../types'
import { apiErrorText } from '../../api/apiErrorText'
import { SettingsSection, inputClass, labelClass, primaryButtonClass } from './settingsSectionUi'

interface PostNLAccountSectionProps {
  account: PostNLAccount
  // Rejects on failure so the section can show why.
  onSave: (input: UpdatePostNLAccountInput) => Promise<void>
}

// Customer code/number, sandbox switch and an optional new API key (empty
// keeps the stored one -- the key itself is never shown).
export function PostNLAccountSection({ account, onSave }: PostNLAccountSectionProps) {
  const { t } = useTranslation(['shipping', 'integrations', 'common'])
  const [customerCode, setCustomerCode] = useState(account.customer_code)
  const [customerNumber, setCustomerNumber] = useState(account.customer_number)
  const [sandbox, setSandbox] = useState(account.sandbox)
  const [apiKey, setApiKey] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      await onSave({ api_key: apiKey, customer_code: customerCode.trim(), customer_number: customerNumber.trim(), sandbox })
      setApiKey('')
      setSaved(true)
    } catch (err) {
      setError(apiErrorText(err) ?? t('postnlSettings.accountSaveFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <SettingsSection id="postnl-account" title={t('postnlSettings.account')} body={t('postnlSettings.accountBody')}>
      <form onSubmit={handleSubmit} className="space-y-3">
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="postnl-settings-customer-code" className={labelClass}>{t('integrations:postnl.customerCode')}</label>
            <input id="postnl-settings-customer-code" type="text" required maxLength={4} value={customerCode}
              onChange={e => setCustomerCode(e.target.value.toUpperCase())} className={inputClass} />
          </div>
          <div>
            <label htmlFor="postnl-settings-customer-number" className={labelClass}>{t('integrations:postnl.customerNumber')}</label>
            <input id="postnl-settings-customer-number" type="text" required maxLength={10} inputMode="numeric" value={customerNumber}
              onChange={e => setCustomerNumber(e.target.value)} className={inputClass} />
          </div>
        </div>
        <div>
          <label htmlFor="postnl-settings-api-key" className={labelClass}>{t('postnlSettings.newApiKey')}</label>
          <input id="postnl-settings-api-key" type="password" autoComplete="new-password" value={apiKey}
            onChange={e => setApiKey(e.target.value)} aria-describedby="postnl-settings-api-key-hint" className={inputClass} />
          <p id="postnl-settings-api-key-hint" className="text-2xs text-ink-4 mt-1">{t('postnlSettings.newApiKeyHint')}</p>
        </div>
        <label className="flex items-center gap-2 text-xs text-ink-2">
          <input type="checkbox" checked={sandbox} onChange={e => setSandbox(e.target.checked)} />
          {t('integrations:postnl.sandbox')}
        </label>
        {error && <p role="alert" className="text-xs text-danger">{error}</p>}
        {saved && <p role="status" className="text-xs text-success">{t('postnlSettings.accountSaved')}</p>}
        <button type="submit" disabled={saving} className={primaryButtonClass}>
          {saving ? t('common:actions.saving') : t('postnlSettings.saveAccount')}
        </button>
      </form>
    </SettingsSection>
  )
}
