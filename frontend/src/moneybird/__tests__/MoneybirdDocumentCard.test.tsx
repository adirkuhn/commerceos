// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MoneybirdDocumentCard } from '../MoneybirdDocumentCard'
import type { UseMoneybirdDocument } from '../hooks/useMoneybirdDocument'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 't' }) }))
vi.mock('../hooks/useMoneybirdDocument', () => ({ useMoneybirdDocument: vi.fn() }))
import { useMoneybirdDocument } from '../hooks/useMoneybirdDocument'

function doc(over: Partial<UseMoneybirdDocument> = {}): UseMoneybirdDocument {
  return {
    connected: true, loading: false, error: null, sending: false, sendError: null, send: vi.fn().mockResolvedValue(undefined),
    state: { document: null, administration_id: 'adm-1', auto_sync: false }, ...over,
  }
}

beforeEach(() => { vi.clearAllMocks() })

describe('MoneybirdDocumentCard', () => {
  it('shows nothing for a shop without Moneybird', () => {
    vi.mocked(useMoneybirdDocument).mockReturnValue(doc({ connected: false, state: null }))
    const { container } = render(<MoneybirdDocumentCard sourceType="sales_order" sourceId="so-1" />)
    expect(container).toBeEmptyDOMElement()
  })

  it('a new order can be sent', async () => {
    const d = doc()
    vi.mocked(useMoneybirdDocument).mockReturnValue(d)
    render(<MoneybirdDocumentCard sourceType="sales_order" sourceId="so-1" />)

    expect(screen.getByText('Not sent to Moneybird yet.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Send to Moneybird' }))
    expect(d.send).toHaveBeenCalled()
  })

  it('a booked, paid order links to its invoice and offers nothing more to send', () => {
    vi.mocked(useMoneybirdDocument).mockReturnValue(doc({ state: { administration_id: 'adm-1', auto_sync: false, document: {
      source_type: 'sales_order', source_id: 'so-1', moneybird_id: 'inv-9', reference: '1234', status: 'synced',
      payment_registered: true, error: '', synced_at: '2026-10-29T10:00:00Z',
    } } }))
    render(<MoneybirdDocumentCard sourceType="sales_order" sourceId="so-1" />)

    expect(screen.getByText('Booked in Moneybird, payment registered.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Open in Moneybird' })).toHaveAttribute('href', 'https://moneybird.com/adm-1/external_sales_invoices/inv-9')
    expect(screen.queryByRole('button', { name: /Moneybird/ })).not.toBeInTheDocument()
  })

  it('a sent quote can be updated and links to its estimate', () => {
    vi.mocked(useMoneybirdDocument).mockReturnValue(doc({ state: { administration_id: 'adm-1', auto_sync: true, document: {
      source_type: 'quote', source_id: 'q-1', moneybird_id: 'est-3', reference: 'Q-0007', status: 'synced',
      payment_registered: false, error: '', synced_at: '2026-10-29T10:00:00Z',
    } } }))
    render(<MoneybirdDocumentCard sourceType="quote" sourceId="q-1" />)

    expect(screen.getByText('Sent to Moneybird as an estimate.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Open in Moneybird' })).toHaveAttribute('href', 'https://moneybird.com/adm-1/estimates/est-3')
    expect(screen.getByRole('button', { name: 'Update in Moneybird' })).toBeInTheDocument()
    expect(screen.getByText('Auto sync is on.')).toBeInTheDocument()
  })

  it('shows why the last send failed, and the send error', () => {
    vi.mocked(useMoneybirdDocument).mockReturnValue(doc({
      sendError: 'Moneybird already has an invoice with this number. It was not booked again.',
      state: { administration_id: 'adm-1', auto_sync: false, document: {
        source_type: 'sales_order', source_id: 'so-1', moneybird_id: '', reference: '1234', status: 'failed',
        payment_registered: false, error: 'tax rate 6.00% not mapped', synced_at: '2026-10-29T10:00:00Z',
      } },
    }))
    render(<MoneybirdDocumentCard sourceType="sales_order" sourceId="so-1" />)

    expect(screen.getByText(/Last send failed: tax rate 6.00% not mapped/)).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('It was not booked again.')
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument()
  })

  it('a booked order gets its payment sent once it is paid', () => {
    const state = { administration_id: 'adm-1', auto_sync: false, document: {
      source_type: 'sales_order' as const, source_id: 'so-1', moneybird_id: 'inv-9', reference: '1234', status: 'synced' as const,
      payment_registered: false, error: '', synced_at: '2026-10-29T10:00:00Z',
    } }
    vi.mocked(useMoneybirdDocument).mockReturnValue(doc({ state }))
    const { rerender } = render(<MoneybirdDocumentCard sourceType="sales_order" sourceId="so-1" />)
    expect(screen.getByText('Booked in Moneybird.')).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()

    rerender(<MoneybirdDocumentCard sourceType="sales_order" sourceId="so-1" orderPaid />)
    expect(screen.getByRole('button', { name: 'Register payment in Moneybird' })).toBeInTheDocument()
  })

  it('a paid order without a payment method was booked without its payment', () => {
    vi.mocked(useMoneybirdDocument).mockReturnValue(doc({ state: { administration_id: 'adm-1', auto_sync: false, document: {
      source_type: 'sales_order', source_id: 'so-1', moneybird_id: 'inv-9', reference: 'CO-1', status: 'synced',
      payment_registered: false, error: '', synced_at: '2026-10-29T10:00:00Z',
    } } }))
    render(<MoneybirdDocumentCard sourceType="sales_order" sourceId="so-1" orderPaid orderPaymentMethod="" />)

    expect(screen.getByText('Booked in Moneybird. Payment not registered: the order has no payment method.')).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('a blocked send says why', () => {
    vi.mocked(useMoneybirdDocument).mockReturnValue(doc())
    render(<MoneybirdDocumentCard sourceType="quote" sourceId="q-1" blockedReason="Save the quote first" />)
    const button = screen.getByRole('button', { name: 'Send to Moneybird' })
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute('title', 'Save the quote first')
  })

  it('disables the button while sending', () => {
    vi.mocked(useMoneybirdDocument).mockReturnValue(doc({ sending: true }))
    render(<MoneybirdDocumentCard sourceType="sales_order" sourceId="so-1" />)
    expect(screen.getByRole('button', { name: 'Sending…' })).toBeDisabled()
  })
})
