import i18next from 'i18next'
import { useCallback, useState } from 'react'
import { apiErrorText } from '../../api/apiErrorText'
import { cancelAssistantRequest, createAssistantRequest, replyToAssistantRequest } from '../api'
import type { AssistantRequest } from '../types'

export interface AssistantActions {
  create: (text: string) => Promise<AssistantRequest | null>
  reply: (id: string, text: string) => Promise<AssistantRequest | null>
  cancel: (id: string) => Promise<AssistantRequest | null>
  busy: boolean
  error: string | null
}

// The assistant's writes. Each resolves to the updated request, or null
// after a failure whose text is in `error` (shown by the screen).
export function useAssistantActions(token: string): AssistantActions {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const run = useCallback(async (call: () => Promise<AssistantRequest>): Promise<AssistantRequest | null> => {
    setBusy(true)
    setError(null)
    try {
      return await call()
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('ui:errors.unknown'))
      return null
    } finally {
      setBusy(false)
    }
  }, [])

  const create = useCallback((text: string) => run(() => createAssistantRequest(token, text)), [run, token])
  const reply = useCallback((id: string, text: string) => run(() => replyToAssistantRequest(token, id, text)), [run, token])
  const cancel = useCallback((id: string) => run(() => cancelAssistantRequest(token, id)), [run, token])

  return { create, reply, cancel, busy, error }
}
