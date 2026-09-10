import { promotionRepository } from '@/repositories'

export const getCoupon = (code: string) => promotionRepository.getCoupon(code)
export const listActiveFlashSales = () => promotionRepository.activeFlashSales()
