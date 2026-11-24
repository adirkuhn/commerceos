import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import { providerLabel } from '../../webhooks/providerLabel'
import { SHIPPING_PROVIDERS, shippingProviderCapabilities } from '../shippingProviders'
import { usePostNLAccount } from './usePostNLAccount'
import { useLabelDefaults } from './useLabelDefaults'
import { PostNLAccountSection } from './PostNLAccountSection'
import { PostNLLabelDefaultsSection } from './PostNLLabelDefaultsSection'

interface PostNLSettingsPageProps {
  connectionId: string
  onClose: () => void
}

// A PostNL connection's settings (/integrations/<connectionId>): its
// account (customer code/number, sandbox, API key) and the defaults new
// labels start with.
export function PostNLSettingsPage({ connectionId, onClose }: PostNLSettingsPageProps) {
  const { token } = useAuth()
  const { t } = useTranslation(['shipping', 'integrations', 'common'])
  const account = usePostNLAccount(token ?? '', connectionId)
  const labelDefaults = useLabelDefaults(token ?? '', connectionId)
  const { packageTypeIds } = shippingProviderCapabilities(SHIPPING_PROVIDERS.postnl)

  return (
    <div className="flex flex-col">
      <div className="flex items-center gap-3 px-5 py-4 border-b border-line">
        <button onClick={onClose} aria-label={t('common:actions.back')} className="p-1.5 rounded-lg hover:bg-hover transition-colors">
          <ArrowLeft size={18} className="text-ink-3" />
        </button>
        <h2 className="text-sm font-semibold text-ink">{providerLabel(SHIPPING_PROVIDERS.postnl)}</h2>
        {account.account?.sandbox && (
          <span data-testid="sandbox-badge" className="text-2xs px-2 py-0.5 rounded-full bg-amber-50 dark:bg-warning/10 text-warning font-medium">
            {t('integrations:postnl.sandbox')}
          </span>
        )}
      </div>
      <div className="p-5 max-w-2xl space-y-8">
        {account.loading && <p className="text-xs text-ink-4">{t('postnlSettings.loading')}</p>}
        {account.error && <p role="alert" className="text-xs text-danger">{account.error}</p>}
        {account.account && <PostNLAccountSection account={account.account} onSave={account.save} />}

        {labelDefaults.error && <p role="alert" className="text-xs text-danger">{labelDefaults.error}</p>}
        {!labelDefaults.loading && !labelDefaults.error && (
          <PostNLLabelDefaultsSection settings={labelDefaults.settings} packageTypeIds={packageTypeIds} onSave={labelDefaults.save} />
        )}
      </div>
    </div>
  )
}
