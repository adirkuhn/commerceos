import { useTranslation } from 'react-i18next'
import type { EmailConnectFormState } from '../hooks/useConnectForms'
import { ConnectFormFooter } from './ConnectFormFooter'

interface Props {
  form: EmailConnectFormState
  error: string
  starting: boolean
}

export function EmailConnectForm({ form, error, starting }: Props) {
  const { t } = useTranslation('integrations')
  return (
    <form onSubmit={form.connect} className="space-y-3">
      <div>
        <label htmlFor="email-host" className="block text-2xs text-ink-4 caps-label mb-1.5">
          {t('email.host')}
        </label>
        <input
          id="email-host"
          type="text"
          required
          autoFocus
          value={form.host}
          onChange={e => form.setHost(e.target.value)}
          placeholder={'smtp.example.com'}
          className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
      </div>
      <div>
        <label htmlFor="email-port" className="block text-2xs text-ink-4 caps-label mb-1.5">
          {t('email.port')}
        </label>
        <input
          id="email-port"
          type="text"
          required
          value={form.port}
          onChange={e => form.setPort(e.target.value)}
          placeholder="587"
          className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
      </div>
      <div>
        <label htmlFor="email-username" className="block text-2xs text-ink-4 caps-label mb-1.5">
          {t('email.username')}
        </label>
        <input
          id="email-username"
          type="text"
          value={form.username}
          onChange={e => form.setUsername(e.target.value)}
          className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
      </div>
      <div>
        <label htmlFor="email-password" className="block text-2xs text-ink-4 caps-label mb-1.5">
          {t('email.password')}
        </label>
        <input
          id="email-password"
          type="password"
          value={form.password}
          onChange={e => form.setPassword(e.target.value)}
          className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
        <p className="mt-1.5 text-2xs text-ink-4">
          {t('email.noAuthHint')}
        </p>
      </div>
      <div>
        <label htmlFor="email-from" className="block text-2xs text-ink-4 caps-label mb-1.5">
          {t('email.from')}
        </label>
        <input
          id="email-from"
          type="email"
          required
          value={form.from}
          onChange={e => form.setFrom(e.target.value)}
          placeholder={'orders@mystore.com'}
          className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
      </div>
      <ConnectFormFooter error={error} starting={starting} />
    </form>
  )
}
