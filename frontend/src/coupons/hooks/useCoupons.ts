import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { Coupon, CouponInput } from '../types'
import * as api from '../api'

interface UseCouponsState {
  coupons: Coupon[]
  loading: boolean
  error: string | null
}

interface UseCouponsActions {
  refresh: () => void
  create: (input: CouponInput) => Promise<Coupon>
  update: (id: string, input: CouponInput) => Promise<Coupon>
  remove: (id: string) => Promise<void>
}

export function useCoupons(token: string): UseCouponsState & UseCouponsActions {
  const [coupons, setCoupons] = useState<Coupon[]>([])
  // true from the first render when a load is coming, so callers never
  // mistake "not fetched yet" for "empty" (e.g. a URL-addressed record
  // flashing "not found" before the list arrives).
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listCoupons(token)
      setCoupons(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('common:status.error'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (input: CouponInput): Promise<Coupon> => {
    const coupon = await api.createCoupon(token, input)
    setCoupons(prev => [...prev, coupon])
    return coupon
  }, [token])

  const update = useCallback(async (id: string, input: CouponInput): Promise<Coupon> => {
    const coupon = await api.updateCoupon(token, id, input)
    setCoupons(prev => prev.map(c => (c.id === id ? coupon : c)))
    return coupon
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteCoupon(token, id)
    setCoupons(prev => prev.filter(c => c.id !== id))
  }, [token])

  return { coupons, loading, error, refresh: load, create, update, remove }
}
