import i18next from 'i18next'
import { useState } from 'react'
import { connectPostNL } from '../../shipping/api'
import { apiErrorText } from '../../api/apiErrorText'

export interface PostNLConnectFormState {
  apiKey: string
  setApiKey: (v: string) => void
  customerCode: string
  setCustomerCode: (v: string) => void
  customerNumber: string
  setCustomerNumber: (v: string) => void
  sandbox: boolean
  setSandbox: (v: boolean) => void
  error: string
  starting: boolean
  connect: (e: React.FormEvent) => Promise<void>
}

// The PostNL connect form's fields and submit. `onConnected` reloads the
// connection list; `onDone` closes the catalog once connected.
export function usePostNLConnectForm(token: string, onConnected: () => void, onDone: () => void): PostNLConnectFormState {
  const [apiKey, setApiKey] = useState('')
  const [customerCode, setCustomerCode] = useState('')
  const [customerNumber, setCustomerNumber] = useState('')
  const [sandbox, setSandbox] = useState(false)
  const [error, setError] = useState('')
  const [starting, setStarting] = useState(false)

  const connect = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token) return
    setError('')
    setStarting(true)
    try {
      await connectPostNL(token, { api_key: apiKey, customer_code: customerCode, customer_number: customerNumber, sandbox })
      setStarting(false)
      onDone()
      onConnected()
    } catch (err) {
      setError(apiErrorText(err) ?? i18next.t('integrations:connect.failed'))
      setStarting(false)
    }
  }

  return {
    apiKey, setApiKey, customerCode, setCustomerCode, customerNumber, setCustomerNumber, sandbox, setSandbox,
    error, starting, connect,
  }
}
