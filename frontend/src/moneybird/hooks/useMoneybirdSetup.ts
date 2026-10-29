import i18next from 'i18next'
import { useCallback, useEffect, useState } from 'react'
import * as api from '../api'
import type { MoneybirdSetup, UpdateMoneybirdSettingsInput } from '../types'
import { apiErrorText } from '../../api/apiErrorText'

export interface UseMoneybirdSetup {
  setup: MoneybirdSetup | null
  loading: boolean
  error: string | null
  // Rejects on failure so the section can show why.
  save: (input: UpdateMoneybirdSettingsInput) => Promise<void>
}

// The Moneybird settings with the administrations, tax rates and ledger
// accounts they choose from. A save reloads it all: choosing an
// administration changes the lists.
export function useMoneybirdSetup(token: string): UseMoneybirdSetup {
  const [setup, setSetup] = useState<MoneybirdSetup | null>(null)
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      setSetup(await api.getMoneybirdSetup(token))
    } catch (err) {
      setError(apiErrorText(err) ?? i18next.t('integrations:moneybird.loadFailed'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { void load() }, [load])

  const save = useCallback(async (input: UpdateMoneybirdSettingsInput) => {
    await api.updateMoneybirdSettings(token, input)
    await load()
  }, [token, load])

  return { setup, loading, error, save }
}
