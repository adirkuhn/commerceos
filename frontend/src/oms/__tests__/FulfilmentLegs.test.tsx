// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { FulfilmentLegs, MiniLegs } from '../FulfilmentLegs'
import type { LegOrder } from '../fulfilmentLegs'

function order(over: Partial<LegOrder> = {}): LegOrder {
  return {
    status: 'processing', payment_status: 'paid', fulfilled: false, packed: false,
    date_paid: '2026-11-24T08:17:00Z', date_completed: null, ...over,
  }
}

describe('FulfilmentLegs', () => {
  it('lists the four legs in order as a labelled list', () => {
    render(<FulfilmentLegs order={order()} />)
    const list = screen.getByRole('list', { name: 'Order journey' })
    const legs = within(list).getAllByRole('listitem')
    expect(legs.map(l => within(l).getByRole('heading').textContent)).toEqual(['Paid', 'Picked', 'Packed', 'Shipped'])
  })

  it('marks the current leg as the current step', () => {
    render(<FulfilmentLegs order={order()} />)
    const current = screen.getAllByRole('listitem').find(l => l.getAttribute('aria-current') === 'step')
    expect(current && within(current).getByRole('heading').textContent).toBe('Picked')
    expect(within(current!).getByText('Up next')).toBeInTheDocument()
  })

  it('stamps done legs and leaves upcoming ones waiting', () => {
    render(<FulfilmentLegs order={order()} />)
    const [paid, , , shipped] = screen.getAllByRole('listitem')
    expect(within(paid).getByText('Done', { exact: false })).toBeInTheDocument()
    expect(within(shipped).getByText('Not yet')).toBeInTheDocument()
  })

  it('shows a packed order as picked too', () => {
    render(<FulfilmentLegs order={order({ packed: true, fulfilled: false })} />)
    const [, picked, packed, shipped] = screen.getAllByRole('listitem')
    expect(within(picked).getByText('Done', { exact: false })).toBeInTheDocument()
    expect(within(packed).getByText('Done', { exact: false })).toBeInTheDocument()
    expect(shipped).toHaveAttribute('aria-current', 'step')
  })

  it('stamps a cancelled order VOID, with no current leg', () => {
    render(<FulfilmentLegs order={order({ status: 'cancelled' })} />)
    expect(screen.getByText('Void')).toBeInTheDocument()
    expect(screen.getAllByRole('listitem').some(l => l.hasAttribute('aria-current'))).toBe(false)
  })
})

describe('MiniLegs', () => {
  it('names where the order is in one accessible label', () => {
    render(<MiniLegs order={order()} />)
    expect(screen.getByRole('img', { name: 'Order journey: Picked up next' })).toBeInTheDocument()
  })

  it('says void for a cancelled order and done when every leg is', () => {
    const { rerender } = render(<MiniLegs order={order({ status: 'cancelled' })} />)
    expect(screen.getByRole('img', { name: 'Order journey: Void' })).toBeInTheDocument()
    rerender(<MiniLegs order={order({ status: 'completed' })} />)
    expect(screen.getByRole('img', { name: 'Order journey: Shipped' })).toBeInTheDocument()
  })
})
