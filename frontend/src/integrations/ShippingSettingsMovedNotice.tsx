import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { SETTINGS_PATH } from '../settings/settingsPaths'

// The default shipping provider and shipping addresses live in
// Settings -> Shipping; this points there from Integrations.
export function ShippingSettingsMovedNotice() {
  const { t } = useTranslation('integrations')
  return (
    <p className="text-2xs text-ink-4 shrink-0">
      {t('shippingMoved.text')}{' '}
      <Link to={`${SETTINGS_PATH}/shipping`} className="text-accent hover:underline">{t('shippingMoved.link')}</Link>
    </p>
  )
}
