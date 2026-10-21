import { useTranslation } from 'react-i18next'
import { usePostNLConnectForm } from '../hooks/usePostNLConnectForm'
import { ConnectFormFooter } from './ConnectFormFooter'

interface Props {
  token: string
  onConnected: () => void
  onDone: () => void
}

const inputClass = 'w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'
const labelClass = 'block text-2xs text-ink-4 caps-label mb-1.5'

export function PostNLConnectForm({ token, onConnected, onDone }: Props) {
  const { t } = useTranslation('integrations')
  const form = usePostNLConnectForm(token, onConnected, onDone)
  return (
    <form onSubmit={form.connect} className="space-y-3">
      <p className="text-2xs text-ink-4">{t('postnl.accountHint')}</p>
      <div>
        <label htmlFor="postnl-api-key" className={labelClass}>{t('postnl.apiKey')}</label>
        <input id="postnl-api-key" type="password" required autoFocus value={form.apiKey}
          onChange={e => form.setApiKey(e.target.value)} className={inputClass} />
      </div>
      <div className="grid grid-cols-2 gap-2">
        <div>
          <label htmlFor="postnl-customer-code" className={labelClass}>{t('postnl.customerCode')}</label>
          <input id="postnl-customer-code" type="text" required maxLength={4} value={form.customerCode}
            onChange={e => form.setCustomerCode(e.target.value.toUpperCase())} placeholder="ABCD" className={inputClass} />
        </div>
        <div>
          <label htmlFor="postnl-customer-number" className={labelClass}>{t('postnl.customerNumber')}</label>
          <input id="postnl-customer-number" type="text" required maxLength={10} inputMode="numeric" value={form.customerNumber}
            onChange={e => form.setCustomerNumber(e.target.value)} className={inputClass} />
        </div>
      </div>
      <label className="flex items-center gap-2 text-xs text-ink-2">
        <input type="checkbox" checked={form.sandbox} onChange={e => form.setSandbox(e.target.checked)} />
        {t('postnl.sandbox')}
      </label>
      <p className="text-2xs text-ink-4">{t('postnl.senderHint')}</p>
      <ConnectFormFooter error={form.error} starting={form.starting} />
    </form>
  )
}
