import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { Customer, CustomerInput } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseCustomersState {
  customers: Customer[]
  loading: boolean
  error: string | null
}

interface UseCustomersActions {
  refresh: () => void
  create: (input: CustomerInput) => Promise<Customer>
  update: (id: string, input: CustomerInput) => Promise<Customer>
  remove: (id: string) => Promise<void>
}

export function useCustomers(token: string): UseCustomersState & UseCustomersActions {
  const [customers, setCustomers] = useState<Customer[]>([])
  // true from the first render when a load is coming -- see useConnections.
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listCustomers(token)
      setCustomers(data)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('customers:errors.unknown'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (input: CustomerInput): Promise<Customer> => {
    const customer = await api.createCustomer(token, input)
    setCustomers(prev => [...prev, customer])
    return customer
  }, [token])

  const update = useCallback(async (id: string, input: CustomerInput): Promise<Customer> => {
    const customer = await api.updateCustomer(token, id, input)
    setCustomers(prev => prev.map(c => (c.id === id ? customer : c)))
    return customer
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteCustomer(token, id)
    setCustomers(prev => prev.filter(c => c.id !== id))
  }, [token])

  return { customers, loading, error, refresh: load, create, update, remove }
}
