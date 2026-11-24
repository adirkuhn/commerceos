import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { ShippingAddress, ShippingAddressInput } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

// A customer's shipping addresses, oldest first, one the default. Adding,
// editing and removing reload the list -- the server may move the default
// (a first address, one added as default, removing the default).
// add/update/remove/makeDefault throw on failure so the form can say why.
export function useShippingAddresses(token: string, customerId: string | null) {
  const [addresses, setAddresses] = useState<ShippingAddress[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !customerId) {
      setAddresses([])
      return
    }
    setLoading(true)
    setError(null)
    try {
      setAddresses(await api.listShippingAddresses(token, customerId))
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('customers:errors.unknown'))
    } finally {
      setLoading(false)
    }
  }, [token, customerId])

  useEffect(() => { void load() }, [load])

  const add = useCallback(async (input: ShippingAddressInput) => {
    if (!customerId) return
    await api.addShippingAddress(token, customerId, input)
    await load()
  }, [token, customerId, load])

  const update = useCallback(async (id: string, input: ShippingAddressInput) => {
    if (!customerId) return
    await api.updateShippingAddress(token, customerId, id, input)
    await load()
  }, [token, customerId, load])

  const remove = useCallback(async (id: string) => {
    if (!customerId) return
    await api.removeShippingAddress(token, customerId, id)
    await load()
  }, [token, customerId, load])

  const makeDefault = useCallback(async (id: string) => {
    if (!customerId) return
    setAddresses(await api.setDefaultShippingAddress(token, customerId, id))
  }, [token, customerId])

  return { addresses, loading, error, add, update, remove, makeDefault }
}
