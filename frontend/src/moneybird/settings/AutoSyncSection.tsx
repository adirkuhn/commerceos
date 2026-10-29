import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { SettingsSection } from '../../shipping/postnl/settingsSectionUi'
import type { MoneybirdSetup } from '../types'
import { useSectionSave, type SaveSettings } from './useSectionSave'

// The auto-sync switch, saved as soon as it's flipped. It flips right away
// and goes back if the save fails.
export function AutoSyncSection({ setup, onSave }: { setup: MoneybirdSetup; onSave: SaveSettings }) {
  const { t } = useTranslation('integrations')
  const [checked, setChecked] = useState(setup.settings.auto_sync)
  const save = useSectionSave(async input => {
    try {
      await onSave(input)
    } catch (err) {
      setChecked(!input.auto_sync)
      throw err
    }
  })
  return (
    <SettingsSection id="mb-auto" title={t('moneybird.auto.title')} body={t('moneybird.auto.body')}>
      <label className="flex items-center gap-2 text-xs text-ink-2">
        <input type="checkbox" checked={checked} disabled={save.saving}
          onChange={e => { setChecked(e.target.checked); void save.run({ auto_sync: e.target.checked }) }} />
        {t('moneybird.auto.label')}
      </label>
      {save.error && <p role="alert" className="text-2xs text-danger">{save.error}</p>}
    </SettingsSection>
  )
}
