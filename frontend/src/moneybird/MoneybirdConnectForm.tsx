import i18next from 'i18next'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { connectMoneybird } from './api'
import { apiErrorText } from '../api/apiErrorText'
import { ConnectFormFooter } from '../integrations/connectForms/ConnectFormFooter'

interface Props {
  token: string
  onConnected: () => void
  onDone: () => void
}

const inputClass = 'w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'
const labelClass = 'block text-2xs text-ink-4 caps-label mb-1.5'

// Connects Moneybird with a personal API token, checked by the backend
// against Moneybird before it is stored.
export function MoneybirdConnectForm({ token, onConnected, onDone }: Props) {
  const { t } = useTranslation('integrations')
  const [apiToken, setApiToken] = useState('')
  const [error, setError] = useState('')
  const [starting, setStarting] = useState(false)

  const connect = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setStarting(true)
    try {
      await connectMoneybird(token, apiToken.trim())
      onDone()
      onConnected()
    } catch (err) {
      setError(apiErrorText(err) ?? i18next.t('integrations:connect.failed'))
    } finally {
      setStarting(false)
    }
  }

  return (
    <form onSubmit={connect} className="space-y-3">
      <p className="text-2xs text-ink-4">{t('moneybird.tokenHint')}</p>
      <div>
        <label htmlFor="moneybird-token" className={labelClass}>{t('moneybird.tokenLabel')}</label>
        <input id="moneybird-token" type="password" required autoFocus value={apiToken}
          onChange={e => setApiToken(e.target.value)} className={inputClass} />
      </div>
      <ConnectFormFooter error={error} starting={starting} />
    </form>
  )
}
