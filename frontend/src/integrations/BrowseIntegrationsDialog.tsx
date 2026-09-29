import type { ReactNode } from 'react'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { Dialog } from '../components/common/Dialog'
import type { ConnectFormProvider, NewPageProvider } from './hooks/useIntegrationsRoute'
import { useConnectForms, type ConnectForms } from './hooks/useConnectForms'
import { WooConnectForm } from './connectForms/WooConnectForm'
import { WhatsAppConnectForm } from './connectForms/WhatsAppConnectForm'
import { StripeConnectForm, RevolutConnectForm } from './connectForms/PaymentConnectForms'
import { EmailConnectForm } from './connectForms/EmailConnectForm'

type CatalogTarget = { connect: ConnectFormProvider } | { new: NewPageProvider }

interface CatalogEntry {
  label: string
  initial: string
  gradient: string
  target: CatalogTarget
}

// What the "Browse integrations" catalog lists, in order. Connect-form
// providers open their form in this dialog; bol/MyParcel open their own
// blank settings page.
export const INTEGRATIONS_CATALOG: CatalogEntry[] = [
  { label: 'WooCommerce', initial: 'W', gradient: 'from-purple-500 to-violet-600', target: { connect: 'woocommerce' } },
  { label: 'Bol.com', initial: 'B', gradient: 'from-blue-500 to-cyan-600', target: { new: 'bol' } },
  { label: 'MyParcel', initial: 'M', gradient: 'from-orange-500 to-red-600', target: { new: 'myparcel' } },
  { label: 'WhatsApp', initial: 'WA', gradient: 'from-green-500 to-emerald-600', target: { connect: 'whatsapp' } },
  { label: 'Stripe', initial: 'S', gradient: 'from-indigo-500 to-purple-600', target: { connect: 'stripe' } },
  { label: 'Revolut', initial: 'R', gradient: 'from-slate-700 to-black', target: { connect: 'revolut' } },
  { get label() { return i18next.t('integrations:browse.email') }, initial: 'E', gradient: 'from-teal-500 to-cyan-600', target: { connect: 'email' } },
]

type FormRenderer = (forms: ConnectForms, onClose: () => void) => ReactNode

const CONNECT_FORMS: Record<ConnectFormProvider, FormRenderer> = {
  woocommerce: forms => <WooConnectForm form={forms.woo} error={forms.error} starting={forms.starting} />,
  whatsapp: (forms, onClose) => (
    <WhatsAppConnectForm form={forms.whatsapp} error={forms.error} starting={forms.starting} onDone={onClose} />
  ),
  stripe: (forms, onClose) => (
    <StripeConnectForm form={forms.stripe} webhook={forms.paymentWebhook} error={forms.error} starting={forms.starting} onDone={onClose} />
  ),
  revolut: (forms, onClose) => (
    <RevolutConnectForm form={forms.revolut} webhook={forms.paymentWebhook} error={forms.error} starting={forms.starting} onDone={onClose} />
  ),
  email: forms => <EmailConnectForm form={forms.email} error={forms.error} starting={forms.starting} />,
}

interface Props {
  token: string
  // The connect form open in the dialog, or null for the catalog list.
  provider: ConnectFormProvider | null
  onOpenConnect: (provider: ConnectFormProvider) => void
  onOpenNew: (provider: NewPageProvider) => void
  onConnected: () => void
  onClose: () => void
}

// The "Browse integrations" modal: the catalog, or one provider's connect
// form. Form state lives here, so it resets whenever the modal closes.
export function BrowseIntegrationsDialog({ token, provider, onOpenConnect, onOpenNew, onConnected, onClose }: Props) {
  const forms = useConnectForms(token, onConnected, onClose)
  const { t } = useTranslation(['integrations', 'common'])

  const openEntry = (target: CatalogTarget) =>
    'connect' in target ? onOpenConnect(target.connect) : onOpenNew(target.new)

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        className="absolute inset-0 dark:bg-black/60 bg-black/30"
        onClick={onClose}
      />
      <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-ink">{t('browse.title')}</h3>
          <button onClick={onClose} aria-label={t('common:actions.close')} className="p-1 -m-1 rounded hover:bg-hover">
            <X size={14} className="text-ink-4" />
          </button>
        </div>

        {provider ? CONNECT_FORMS[provider](forms, onClose) : (
          <div className="space-y-2">
            {INTEGRATIONS_CATALOG.map(entry => (
              <button
                key={entry.initial}
                onClick={() => openEntry(entry.target)}
                className="w-full flex items-center gap-3 p-3 rounded-lg border border-line hover:bg-hover transition-colors text-left"
              >
                <div className={`w-8 h-8 rounded-lg bg-gradient-to-br ${entry.gradient} flex items-center justify-center text-white font-bold text-xs shrink-0`}>
                  {entry.initial}
                </div>
                <span className="text-sm text-ink">{entry.label}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </Dialog>
  )
}
