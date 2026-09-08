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
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listCoupons(token)
      setCoupons(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
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
