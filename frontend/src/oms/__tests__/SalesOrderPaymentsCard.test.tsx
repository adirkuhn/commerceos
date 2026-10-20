// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { SalesOrderPaymentsCard } from '../SalesOrderPaymentsCard'
import { formatMoney } from '../../format/money'
import type { SalesOrderPayment } from '../types'

const cashPlusCard: SalesOrderPayment[] = [
  { method: 'Cash', amount_minor_units: 500, transaction_id: '' },
  { method: 'Revolut', amount_minor_units: 1500, transaction_id: 'rev-1' },
]

describe('SalesOrderPaymentsCard', () => {
  it('is titled Payments', () => {
    render(<SalesOrderPaymentsCard payments={cashPlusCard} currency="EUR" />)

    expect(screen.getByRole('heading', { name: 'Payments' })).toBeInTheDocument()
  })

  it('lists every payment with its method and amount', () => {
    render(<SalesOrderPaymentsCard payments={cashPlusCard} currency="EUR" />)

    expect(screen.getByText('Cash')).toBeInTheDocument()
    expect(screen.getByText(formatMoney(500, 'EUR'))).toBeInTheDocument()
    expect(screen.getByText('Revolut')).toBeInTheDocument()
    expect(screen.getByText(formatMoney(1500, 'EUR'))).toBeInTheDocument()
  })

  it('shows the transaction id of a payment that has one', () => {
    render(<SalesOrderPaymentsCard payments={cashPlusCard} currency="EUR" />)

    expect(screen.getByText('rev-1')).toBeInTheDocument()
    expect(screen.getAllByText('Transaction:')).toHaveLength(1)
  })

  it('renders nothing when the order has no payments', () => {
    const { container } = render(<SalesOrderPaymentsCard payments={[]} currency="EUR" />)

    expect(container).toBeEmptyDOMElement()
  })
})
