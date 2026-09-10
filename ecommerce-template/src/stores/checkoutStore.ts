import { create } from 'zustand'
import type { Address, ShippingMethodId, PaymentMethodId } from '@/types'

export type CheckoutStep = 'customer' | 'shipping' | 'payment' | 'review'
export const CHECKOUT_STEPS: CheckoutStep[] = ['customer', 'shipping', 'payment', 'review']

interface CheckoutState {
  step: CheckoutStep
  email: string
  billingAddress: Address | null
  shippingAddress: Address | null
  shippingAddressSameAsBilling: boolean
  shippingMethodId: ShippingMethodId
  paymentMethodId: PaymentMethodId
  acceptedTerms: boolean
  goToStep: (step: CheckoutStep) => void
  nextStep: () => void
  setEmail: (email: string) => void
  setBillingAddress: (address: Address) => void
  setShippingAddress: (address: Address) => void
  setShippingAddressSameAsBilling: (same: boolean) => void
  setShippingMethodId: (id: ShippingMethodId) => void
  setPaymentMethodId: (id: PaymentMethodId) => void
  setAcceptedTerms: (accepted: boolean) => void
  reset: () => void
}

const initial = {
  step: 'customer' as CheckoutStep,
  email: '',
  billingAddress: null,
  shippingAddress: null,
  shippingAddressSameAsBilling: true,
  shippingMethodId: 'standard' as ShippingMethodId,
  paymentMethodId: 'card' as PaymentMethodId,
  acceptedTerms: false,
}

export const useCheckoutStore = create<CheckoutState>()((set, get) => ({
  ...initial,
  goToStep: (step) => set({ step }),
  nextStep: () => {
    const index = CHECKOUT_STEPS.indexOf(get().step)
    const next = CHECKOUT_STEPS[Math.min(index + 1, CHECKOUT_STEPS.length - 1)]
    set({ step: next })
  },
  setEmail: (email) => set({ email }),
  setBillingAddress: (billingAddress) => set({ billingAddress }),
  setShippingAddress: (shippingAddress) => set({ shippingAddress }),
  setShippingAddressSameAsBilling: (shippingAddressSameAsBilling) => set({ shippingAddressSameAsBilling }),
  setShippingMethodId: (shippingMethodId) => set({ shippingMethodId }),
  setPaymentMethodId: (paymentMethodId) => set({ paymentMethodId }),
  setAcceptedTerms: (acceptedTerms) => set({ acceptedTerms }),
  reset: () => set(initial),
}))
