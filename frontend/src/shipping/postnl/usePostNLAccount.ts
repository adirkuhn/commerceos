import i18next from 'i18next'
import { useCallback, useEffect, useState } from 'react'
import * as shippingApi from '../api'
import type { PostNLAccount, UpdatePostNLAccountInput } from '../types'
import { apiErrorText } from '../../api/apiErrorText'

export interface UsePostNLAccount {
  account: PostNLAccount | null
  loading: boolean
  error: string | null
  // Rejects on failure so the caller can show why.
  save: (input: UpdatePostNLAccountInput) => Promise<void>
}

// A PostNL connection's customer code/number and sandbox switch (never the
// API key).
export function usePostNLAccount(token: string, connectionId: string): UsePostNLAccount {
  const [account, setAccount] = useState<PostNLAccount | null>(null)
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError(null)
    shippingApi.getPostNLConnection(token, connectionId)
      .then(result => { if (!cancelled) setAccount(result) })
      .catch(err => { if (!cancelled) setError(apiErrorText(err) ?? i18next.t('shipping:postnlSettings.accountLoadFailed')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, connectionId])

  const save = useCallback(async (input: UpdatePostNLAccountInput) => {
    setAccount(await shippingApi.updatePostNLConnection(token, connectionId, input))
  }, [token, connectionId])

  return { account, loading, error, save }
}
