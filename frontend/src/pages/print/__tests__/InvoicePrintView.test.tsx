// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { InvoicePrintView } from '../InvoicePrintView'
import type { SalesOrder } from '../../../oms/types'
import type { CompanyProfile } from '../../../settings/company/types'

const order = {
  id: 'so-1', number: '1001', external_order_id: '', occurred_at: '2026-09-24T10:00:00Z', currency_code: 'EUR',
  total_minor_units: 1000, billing: undefined, shipping: undefined,
} as unknown as SalesOrder

const seller: CompanyProfile = {
  legal_name: 'Cantinho Brasileiro B.V.', trading_name: 'Cantinho', vat_number: 'NL123456789B01', chamber_of_commerce: '12345678',
  email: 'info@example.nl', phone: '+31 20 123 4567', iban: 'NL91ABNA0417164300',
  address: { street: 'Damrak', house_number: '1', postal_code: '1012 LG', city: 'Amsterdam', country: 'NL' },
  updated_at: '',
}

beforeEach(() => { window.print = vi.fn() })

describe('InvoicePrintView seller block', () => {
  it('prints who is selling: legal name, address, VAT, chamber of commerce and IBAN', () => {
    render(<InvoicePrintView order={order} lineItems={[]} seller={seller} onClose={vi.fn()} />)

    const block = screen.getByRole('region', { name: 'Seller' })
    expect(within(block).getByText('Cantinho Brasileiro B.V.')).toBeInTheDocument()
    expect(within(block).getByText('Damrak 1')).toBeInTheDocument()
    expect(within(block).getByText('1012 LG Amsterdam')).toBeInTheDocument()
    expect(within(block).getByText('VAT: NL123456789B01')).toBeInTheDocument()
    expect(within(block).getByText('Chamber of Commerce: 12345678')).toBeInTheDocument()
    expect(within(block).getByText('IBAN: NL91ABNA0417164300')).toBeInTheDocument()
  })

  it('leaves out lines that are empty', () => {
    render(<InvoicePrintView order={order} lineItems={[]} seller={{ ...seller, vat_number: '', iban: '' }} onClose={vi.fn()} />)
    const block = screen.getByRole('region', { name: 'Seller' })
    expect(within(block).queryByText(/VAT:/)).not.toBeInTheDocument()
    expect(within(block).queryByText(/IBAN:/)).not.toBeInTheDocument()
  })

  it('without a company profile, tells staff where to add it (not printed)', () => {
    render(<InvoicePrintView order={order} lineItems={[]} seller={null} onClose={vi.fn()} />)

    expect(screen.queryByRole('region', { name: 'Seller' })).not.toBeInTheDocument()
    const hint = screen.getByText(/Add your company details in Settings/)
    expect(hint.closest('.print\\:hidden')).not.toBeNull()
  })
})

describe('InvoicePrintView customer block', () => {
  it('addresses a company first, then its contact person as Attn.', () => {
    const b2b = { ...order, billing: { company: 'Claus Hoofddorp B.V.', first_name: 'Kylian', last_name: 'Klockenbrink', address_1: 'Bosweg 19', city: 'Hoofddorp' } } as SalesOrder
    render(<InvoicePrintView order={b2b} lineItems={[]} seller={seller} onClose={vi.fn()} />)

    const company = screen.getByText('Claus Hoofddorp B.V.')
    const attn = screen.getByText('Attn. Kylian Klockenbrink')
    expect(company.compareDocumentPosition(attn) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('addresses a private customer by name', () => {
    const b2c = { ...order, billing: { first_name: 'Ana', last_name: 'Souza', city: 'Utrecht' } } as SalesOrder
    render(<InvoicePrintView order={b2c} lineItems={[]} seller={seller} onClose={vi.fn()} />)
    expect(screen.getByText('Ana Souza')).toBeInTheDocument()
    expect(screen.queryByText(/Attn\./)).not.toBeInTheDocument()
  })
})
