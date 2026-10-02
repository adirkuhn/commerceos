import { describe, it, expect } from 'vitest'
import { PAYMENT_STATUSES, paymentStatusLabel, paymentStatusTone } from '../paymentStatus'

describe('paymentStatusTone', () => {
  it('is good for paid, warning while waiting, bad when failed, neutral otherwise', () => {
    expect(paymentStatusTone('paid')).toBe('success')
    expect(paymentStatusTone('awaiting_payment')).toBe('warning')
    expect(paymentStatusTone('awaiting_confirmation')).toBe('warning')
    expect(paymentStatusTone('failed')).toBe('danger')
    expect(paymentStatusTone('refunded')).toBe('neutral')
    expect(paymentStatusTone('partially_refunded')).toBe('neutral')
    expect(paymentStatusTone('cancelled_unpaid')).toBe('neutral')
  })
})

describe('paymentStatusLabel', () => {
  it('has an admin-language label for every status', () => {
    for (const s of PAYMENT_STATUSES) {
      expect(paymentStatusLabel(s)).not.toMatch(/paymentStatus\./)
    }
    expect(paymentStatusLabel('paid')).toBe('Paid')
    expect(paymentStatusLabel('awaiting_confirmation')).toBe('Awaiting confirmation')
  })
})
