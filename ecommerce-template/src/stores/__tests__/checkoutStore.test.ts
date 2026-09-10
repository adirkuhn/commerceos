import { describe, expect, it, beforeEach } from 'vitest'
import { useCheckoutStore, CHECKOUT_STEPS } from '../checkoutStore'

beforeEach(() => useCheckoutStore.getState().reset())

describe('useCheckoutStore', () => {
  it('starts on the customer step', () => {
    expect(useCheckoutStore.getState().step).toBe('customer')
  })

  it('advances one step at a time and stops at review', () => {
    for (const _ of CHECKOUT_STEPS) useCheckoutStore.getState().nextStep()
    expect(useCheckoutStore.getState().step).toBe('review')
  })

  it('reset clears everything back to defaults', () => {
    useCheckoutStore.getState().setEmail('a@b.com')
    useCheckoutStore.getState().goToStep('payment')
    useCheckoutStore.getState().reset()
    expect(useCheckoutStore.getState().email).toBe('')
    expect(useCheckoutStore.getState().step).toBe('customer')
  })
})
