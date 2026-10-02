// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { OrderPaymentCard } from '../OrderPaymentCard'
import { PaymentStatusBadge } from '../PaymentStatusBadge'
import type { SalesOrder } from '../types'

function order(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 't1', source: 'woocommerce', external_order_id: '501', number: '501', order_key: 'k',
    status: 'processing', payment_status: 'paid', currency_code: 'EUR', discount_total_minor_units: 0,
    shipping_total_minor_units: 0, tax_total_minor_units: 0, total_minor_units: 1000, occurred_at: '', created_at: '',
    ...overrides,
  }
}

describe('PaymentStatusBadge', () => {
  it('labels the status and colours it by outcome', () => {
    const { rerender } = render(<PaymentStatusBadge status="paid" />)
    expect(screen.getByText('Paid')).toHaveClass('text-success')
    rerender(<PaymentStatusBadge status="awaiting_payment" />)
    expect(screen.getByText('Awaiting payment')).toHaveClass('text-warning')
    rerender(<PaymentStatusBadge status="failed" />)
    expect(screen.getByText('Payment failed')).toHaveClass('text-danger')
  })
})

describe('OrderPaymentCard', () => {
  it('shows whether the payment succeeded, the method, transaction id and when it was paid', () => {
    render(<OrderPaymentCard order={order({
      payment_method: 'stripe', payment_method_title: 'Credit card (Stripe)', payment_reference: 'pi_3Q123',
      date_paid: '2026-10-02T14:31:00Z', shipping_method_title: 'PostNL',
    })} />)

    expect(screen.getByText('Paid')).toBeInTheDocument()
    expect(screen.getByText('Credit card (Stripe)')).toBeInTheDocument()
    expect(screen.getByText('pi_3Q123')).toBeInTheDocument()
    expect(screen.getByText('Paid on:')).toBeInTheDocument()
    expect(screen.getByText('PostNL')).toBeInTheDocument()
  })

  it('copies the transaction id', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    render(<OrderPaymentCard order={order({ payment_reference: 'pi_3Q123' })} />)

    await userEvent.click(screen.getByRole('button', { name: 'Copy transaction id' }))

    expect(writeText).toHaveBeenCalledWith('pi_3Q123')
    expect(await screen.findByText('Copied')).toBeInTheDocument()
  })

  it('leaves out the transaction and paid date when there are none (e.g. awaiting payment)', () => {
    render(<OrderPaymentCard order={order({ status: 'pending', payment_status: 'awaiting_payment', payment_method_title: 'Bank transfer' })} />)

    expect(screen.getByText('Awaiting payment')).toBeInTheDocument()
    expect(screen.queryByText('Transaction:')).not.toBeInTheDocument()
    expect(screen.queryByText('Paid on:')).not.toBeInTheDocument()
  })

  it('names bol.com as the method for a bol order', () => {
    render(<OrderPaymentCard order={order({ source: 'bol', payment_method: 'bol', payment_method_title: 'bol.com' })} />)

    expect(screen.getByText('bol.com')).toBeInTheDocument()
  })
})
