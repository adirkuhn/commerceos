import { useTranslation } from 'react-i18next'
import type { WhatsAppConnectFormState } from '../hooks/useConnectForms'
import { ConnectFormFooter } from './ConnectFormFooter'

interface Props {
  form: WhatsAppConnectFormState
  error: string
  starting: boolean
  onDone: () => void
}

export function WhatsAppConnectForm({ form, error, starting, onDone }: Props) {
  const { t } = useTranslation('integrations')
  if (form.webhookUrl) {
    return (
      <div className="space-y-3">
        <p className="text-xs text-ink-2">
          {t('whatsapp.connectedPasteUrl')}
        </p>
        <p className="text-2xs font-mono break-all bg-fill rounded-lg px-2 py-1.5 text-ink-2">
          {form.webhookUrl}
        </p>
        <button
          onClick={onDone}
          className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
        >
          {t('done')}
        </button>
      </div>
    )
  }

  return (
    <form onSubmit={form.connect} className="space-y-3">
      <div>
        <label className="block text-2xs text-ink-4 caps-label mb-1.5">
          {t('whatsapp.baseUrl')}
        </label>
        <input
          type="url"
          required
          autoFocus
          value={form.baseUrl}
          onChange={e => form.setBaseUrl(e.target.value)}
          placeholder={'https://evolution.example.com'}
          className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
        <p className="mt-1.5 text-2xs text-ink-4">
          {t('whatsapp.baseUrlHint')}
        </p>
      </div>
      <div>
        <label htmlFor="wa-instance-name" className="block text-2xs text-ink-4 caps-label mb-1.5">
          {t('whatsapp.instanceName')}
        </label>
        <input
          id="wa-instance-name"
          type="text"
          required
          value={form.instanceName}
          onChange={e => form.setInstanceName(e.target.value)}
          className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
      </div>
      <div>
        <label htmlFor="wa-api-key" className="block text-2xs text-ink-4 caps-label mb-1.5">
          {t('whatsapp.apiKey')}
        </label>
        <input
          id="wa-api-key"
          type="password"
          required
          value={form.apiKey}
          onChange={e => form.setApiKey(e.target.value)}
          className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
      </div>
      <ConnectFormFooter error={error} starting={starting} />
    </form>
  )
}
