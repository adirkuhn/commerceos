import { describe, it, expect } from 'vitest'
import { fulfilmentLegs, type LegOrder } from '../fulfilmentLegs'

function order(over: Partial<LegOrder> = {}): LegOrder {
  return {
    status: 'processing', payment_status: 'awaiting_payment',
    fulfilled: false, packed: false, date_paid: null, date_completed: null,
    ...over,
  }
}

const states = (o: LegOrder) => fulfilmentLegs(o).legs.map(l => `${l.id}:${l.state}`)

describe('fulfilmentLegs', () => {
  it('runs paid, picked, packed, shipped in that order', () => {
    expect(fulfilmentLegs(order()).legs.map(l => l.id)).toEqual(['paid', 'picked', 'packed', 'shipped'])
  })

  it('makes the first unfinished leg the current one', () => {
    expect(states(order({ payment_status: 'paid' }))).toEqual(['paid:done', 'picked:current', 'packed:upcoming', 'shipped:upcoming'])
  })

  it('counts a packed order as picked too -- picking is optional', () => {
    expect(states(order({ payment_status: 'paid', packed: true, fulfilled: false }))).toEqual(['paid:done', 'picked:done', 'packed:done', 'shipped:current'])
  })

  it('counts a shipped order as picked and packed, even when packed outside CommerceOS', () => {
    expect(states(order({ status: 'shipped', payment_status: 'paid' }))).toEqual(['paid:done', 'picked:done', 'packed:done', 'shipped:done'])
    expect(states(order({ status: 'completed', payment_status: 'paid' }))).toEqual(['paid:done', 'picked:done', 'packed:done', 'shipped:done'])
  })

  it('keeps payment current while the goods move ahead (pay on invoice)', () => {
    expect(states(order({ status: 'shipped' }))).toEqual(['paid:current', 'picked:done', 'packed:done', 'shipped:done'])
  })

  it('treats refunded money as having been paid', () => {
    expect(fulfilmentLegs(order({ payment_status: 'partially_refunded', status: 'partially_refunded' })).legs[0].state).toBe('done')
  })

  it('stamps cancelled, failed and refunded orders void, with no current leg', () => {
    for (const status of ['cancelled', 'failed', 'refunded'] as const) {
      const result = fulfilmentLegs(order({ status, payment_status: 'paid' }))
      expect(result.voided, status).toBe(true)
      expect(result.legs.some(l => l.state === 'current'), status).toBe(false)
      expect(result.legs[0].state, status).toBe('done')
    }
    expect(fulfilmentLegs(order()).voided).toBe(false)
  })

  it('dates the legs it knows a date for', () => {
    const result = fulfilmentLegs(order({ status: 'completed', payment_status: 'paid', date_paid: '2026-11-24T08:17:00Z', date_completed: '2026-11-25T10:00:00Z' }))
    expect(result.legs.find(l => l.id === 'paid')?.at).toBe('2026-11-24T08:17:00Z')
    expect(result.legs.find(l => l.id === 'shipped')?.at).toBe('2026-11-25T10:00:00Z')
    expect(result.legs.find(l => l.id === 'picked')?.at).toBeNull()
  })
})
