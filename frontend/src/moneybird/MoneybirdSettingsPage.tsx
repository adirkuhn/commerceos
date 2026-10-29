import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useTaxRates } from '../tax/hooks/useTaxRates'
import { useMoneybirdSetup } from './hooks/useMoneybirdSetup'
import { AdministrationSection } from './settings/AdministrationSection'
import { VatRatesSection } from './settings/VatRatesSection'
import { PaymentMethodsSection } from './settings/PaymentMethodsSection'
import { AutoSyncSection } from './settings/AutoSyncSection'

// The Moneybird connection's settings (/integrations/<connectionId>):
// administration and ledger account, VAT-rate and payment-method mappings,
// and auto sync. Each section saves only its own fields.
export function MoneybirdSettingsPage({ onClose }: { onClose: () => void }) {
  const { token } = useAuth()
  const { t } = useTranslation(['integrations', 'common'])
  const mb = useMoneybirdSetup(token ?? '')
  const { rates } = useTaxRates(token ?? '')
  const shopRates = [...new Set(rates.map(r => r.rate_basis_points))]
  const setup = mb.setup
  // Sections remount after an administration switch, so they start from the new administration's settings.
  const key = setup ? `${setup.settings.administration_id}` : ''

  return (
    <div className="flex flex-col">
      <div className="flex items-center gap-3 px-5 py-4 border-b border-line">
        <button onClick={onClose} aria-label={t('common:actions.back')} className="p-1.5 rounded-lg hover:bg-hover transition-colors">
          <ArrowLeft size={18} className="text-ink-3" />
        </button>
        <h2 className="text-sm font-semibold text-ink">{t('moneybird.name')}</h2>
        {setup?.settings.administration_name && <span className="text-2xs text-ink-4">{setup.settings.administration_name}</span>}
      </div>
      <div className="p-5 max-w-2xl space-y-8">
        {mb.loading && !setup && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
        {mb.error && <p role="alert" className="text-xs text-danger">{mb.error}</p>}
        {setup && (
          <div key={key} className="space-y-8">
            <AdministrationSection setup={setup} onSave={mb.save} />
            <VatRatesSection setup={setup} shopRates={shopRates} onSave={mb.save} />
            <PaymentMethodsSection setup={setup} onSave={mb.save} />
            <AutoSyncSection setup={setup} onSave={mb.save} />
          </div>
        )}
      </div>
    </div>
  )
}
