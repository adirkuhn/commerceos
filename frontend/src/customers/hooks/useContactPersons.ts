import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { ContactPerson, ContactPersonInput } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

// A customer's contact persons, the main one first. add/update/remove throw
// on failure so the form showing them can say why.
export function useContactPersons(token: string, customerId: string | null) {
  const [persons, setPersons] = useState<ContactPerson[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !customerId) {
      setPersons([])
      return
    }
    setLoading(true)
    setError(null)
    try {
      setPersons(await api.listContactPersons(token, customerId))
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('customers:errors.unknown'))
    } finally {
      setLoading(false)
    }
  }, [token, customerId])

  useEffect(() => { void load() }, [load])

  const add = useCallback(async (input: ContactPersonInput) => {
    if (!customerId) return
    const person = await api.addContactPerson(token, customerId, input)
    setPersons(prev => [...prev, person])
  }, [token, customerId])

  const update = useCallback(async (id: string, input: ContactPersonInput) => {
    if (!customerId) return
    const person = await api.updateContactPerson(token, customerId, id, input)
    setPersons(prev => prev.map(p => (p.id === id ? person : p)))
  }, [token, customerId])

  const remove = useCallback(async (id: string) => {
    if (!customerId) return
    await api.removeContactPerson(token, customerId, id)
    setPersons(prev => prev.filter(p => p.id !== id))
  }, [token, customerId])

  return { persons, loading, error, add, update, remove }
}
