import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { ExchangeRate, UpsertExchangeRateInput } from '../types'
import * as api from '../api'

interface UseExchangeRatesState {
  rates: ExchangeRate[]
  loading: boolean
  error: string | null
}

interface UseExchangeRatesActions {
  refresh: () => void
  upsert: (input: UpsertExchangeRateInput) => Promise<ExchangeRate>
  remove: (currencyCode: string) => Promise<void>
}

export function useExchangeRates(token: string): UseExchangeRatesState & UseExchangeRatesActions {
  const [rates, setRates] = useState<ExchangeRate[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listExchangeRates(token)
      setRates(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('settings:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const upsert = useCallback(async (input: UpsertExchangeRateInput): Promise<ExchangeRate> => {
    const result = await api.upsertExchangeRate(token, input)
    setRates(prev => (prev.some(r => r.currency_code === result.currency_code) ? prev.map(r => (r.currency_code === result.currency_code ? result : r)) : [...prev, result]))
    return result
  }, [token])

  const remove = useCallback(async (currencyCode: string): Promise<void> => {
    await api.deleteExchangeRate(token, currencyCode)
    setRates(prev => prev.filter(r => r.currency_code !== currencyCode))
  }, [token])

  return { rates, loading, error, refresh: load, upsert, remove }
}
