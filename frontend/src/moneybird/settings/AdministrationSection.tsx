import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { inputClass, labelClass, SettingsSection } from '../../shipping/postnl/settingsSectionUi'
import type { MoneybirdSetup } from '../types'
import { SectionFooter } from './SectionFooter'
import { useSectionSave, type SaveSettings } from './useSectionSave'

// Which administration to book in, and the revenue ledger account. A new
// administration is saved on its own first: its ledger accounts and tax
// rates only load once it is chosen.
export function AdministrationSection({ setup, onSave }: { setup: MoneybirdSetup; onSave: SaveSettings }) {
  const { t } = useTranslation('integrations')
  const { settings } = setup
  const [administrationId, setAdministrationId] = useState(settings.administration_id)
  const [ledgerId, setLedgerId] = useState(settings.default_ledger_account_id)
  const save = useSectionSave(onSave)
  const switching = administrationId !== settings.administration_id

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    void save.run(switching ? { administration_id: administrationId } : { default_ledger_account_id: ledgerId })
  }

  return (
    <SettingsSection id="mb-admin" title={t('moneybird.admin.title')} body={t('moneybird.admin.body')}>
      <form onSubmit={submit} className="space-y-3">
        <div>
          <label htmlFor="mb-administration" className={labelClass}>{t('moneybird.admin.administration')}</label>
          <select id="mb-administration" value={administrationId} onChange={e => setAdministrationId(e.target.value)} className={inputClass}>
            <option value="" disabled>{t('moneybird.admin.choose')}</option>
            {setup.administrations.map(a => <option key={a.id} value={a.id}>{a.name} ({a.currency})</option>)}
          </select>
          {switching && settings.administration_id && (
            <p className="mt-1 text-2xs text-warning">{t('moneybird.admin.switchWarning')}</p>
          )}
        </div>
        {!switching && settings.administration_id && (
          <div>
            <label htmlFor="mb-ledger" className={labelClass}>{t('moneybird.admin.ledger')}</label>
            <select id="mb-ledger" value={ledgerId} onChange={e => setLedgerId(e.target.value)} className={inputClass}>
              <option value="" disabled>{t('moneybird.admin.choose')}</option>
              {setup.ledger_accounts.map(l => <option key={l.id} value={l.id}>{l.name}</option>)}
            </select>
          </div>
        )}
        <SectionFooter {...save} disabled={switching ? !administrationId : !ledgerId} />
      </form>
    </SettingsSection>
  )
}
