import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { isTenantAdmin } from '../auth/roles'
import { DefaultShippingProviderSetting } from '../shipping/DefaultShippingProviderSetting'
import { ShippingAddressesSetting } from '../shipping/addresses/ShippingAddressesSetting'
import { useShippingConnections } from '../shipping/useShippingConnections'
import { INTEGRATIONS_PATH } from '../integrations/integrationsPaths'

// Settings -> Shipping: packing's default shipping provider and the sender
// / return addresses on labels. Every role may read; owners and admins
// change them. Without a shipping connection there is nothing to set yet.
export function ShippingSettingsSection() {
  const { token, user } = useAuth()
  const { t } = useTranslation('settings')
  const canEdit = isTenantAdmin(user?.role)
  const shipping = useShippingConnections(token ?? '')

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-base font-semibold text-ink mb-1">{t('sections.shipping')}</h2>
        <p className="text-xs text-ink-4">{t('shipping.subtitle')}</p>
      </div>
      {!shipping.loading && shipping.connections.length === 0 && !shipping.error && (
        <p className="text-xs text-ink-3">
          {t('shipping.noConnection')}{' '}
          <Link to={`${INTEGRATIONS_PATH}/connect`} className="text-accent hover:underline">{t('shipping.connect')}</Link>
        </p>
      )}
      <DefaultShippingProviderSetting token={token ?? ''} canEdit={canEdit} />
      <ShippingAddressesSetting token={token ?? ''} canEdit={canEdit} />
    </div>
  )
}
