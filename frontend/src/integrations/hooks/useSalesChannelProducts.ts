import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { SalesChannelProductEnrollment } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseSalesChannelProductsState {
  items: SalesChannelProductEnrollment[]
  total: number
  loading: boolean
  error: string | null
}

interface UseSalesChannelProductsActions {
  refresh: () => void
  setOverride: (productId: string, included: boolean) => Promise<void>
}

// channelId is null while no channel is selected for management — nothing
// to load until the user opens a specific channel's product picker.
export function useSalesChannelProducts(
  token: string,
  channelId: string | null,
  params: { page?: number; perPage?: number; search?: string } = {},
): UseSalesChannelProductsState & UseSalesChannelProductsActions {
  const { page, perPage, search } = params
  const [items, setItems] = useState<SalesChannelProductEnrollment[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(!!channelId)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !channelId) {
      setItems([])
      setTotal(0)
      return
    }
    setLoading(true)
    setError(null)
    try {
      const result = await api.listSalesChannelProducts(token, channelId, { page, perPage, search })
      setItems(result.items)
      setTotal(result.total)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('integrations:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token, channelId, page, perPage, search])

  useEffect(() => { load() }, [load])

  // Refetches the current page instead of patching the enrolled flag
  // locally -- the flag is server-computed (channel rule + override), so a
  // local patch could drift from what the rule would otherwise say.
  const setOverride = useCallback(async (productId: string, included: boolean): Promise<void> => {
    if (!channelId) return
    await api.setSalesChannelProductOverride(token, channelId, productId, included)
    await load()
  }, [token, channelId, load])

  return { items, total, loading, error, refresh: load, setOverride }
}
