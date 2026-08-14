import { useState, useEffect, useCallback } from 'react'
import type { TaxRate, UpsertTaxRateInput } from '../types'
import * as api from '../api'

interface UseTaxRatesState {
  rates: TaxRate[]
  loading: boolean
  error: string | null
}

interface UseTaxRatesActions {
  refresh: () => void
  upsert: (input: UpsertTaxRateInput) => Promise<TaxRate>
  remove: (id: string) => Promise<void>
}

export function useTaxRates(token: string): UseTaxRatesState & UseTaxRatesActions {
  const [rates, setRates] = useState<TaxRate[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listTaxRates(token)
      setRates(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const upsert = useCallback(async (input: UpsertTaxRateInput): Promise<TaxRate> => {
    const result = await api.upsertTaxRate(token, input)
    setRates(prev => (prev.some(r => r.id === result.id) ? prev.map(r => (r.id === result.id ? result : r)) : [...prev, result]))
    return result
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteTaxRate(token, id)
    setRates(prev => prev.filter(r => r.id !== id))
  }, [token])

  return { rates, loading, error, refresh: load, upsert, remove }
}
