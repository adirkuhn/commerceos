import { useTranslation } from 'react-i18next'
import { primaryButtonClass } from '../../shipping/postnl/settingsSectionUi'

interface Props {
  saving: boolean
  saved: boolean
  error: string | null
  disabled?: boolean
}

// A section's Save button with its outcome next to it.
export function SectionFooter({ saving, saved, error, disabled }: Props) {
  const { t } = useTranslation('integrations')
  return (
    <div className="flex items-center gap-3">
      <button type="submit" disabled={saving || disabled} className={primaryButtonClass}>{t('moneybird.save')}</button>
      {saved && <span role="status" className="text-2xs text-success">{t('moneybird.saved')}</span>}
      {error && <span role="alert" className="text-2xs text-danger">{error}</span>}
    </div>
  )
}
