// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { InvoiceWithSeller } from '../InvoiceWithSeller'
import type { SalesOrder } from '../../../oms/types'

vi.mock('../../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../../../settings/company/api', async importOriginal => {
  const actual = await importOriginal<typeof import('../../../settings/company/api')>()
  return { ...actual, getCompanyProfile: vi.fn() }
})
import * as companyApi from '../../../settings/company/api'

const order = { id: 'so-1', number: '1001', occurred_at: '2026-09-24T10:00:00Z', currency_code: 'EUR' } as unknown as SalesOrder

beforeEach(() => { window.print = vi.fn() })

describe('InvoiceWithSeller', () => {
  it('waits for the company profile before opening the print dialog', async () => {
    let resolve!: (v: Awaited<ReturnType<typeof companyApi.getCompanyProfile>>) => void
    vi.mocked(companyApi.getCompanyProfile).mockReturnValue(new Promise(r => { resolve = r }))
    render(<InvoiceWithSeller order={order} lineItems={[]} onClose={vi.fn()} />)

    expect(screen.getByText('Preparing invoice…')).toBeInTheDocument()
    expect(window.print).not.toHaveBeenCalled()

    resolve({
      legal_name: 'Cantinho B.V.', trading_name: '', vat_number: '', chamber_of_commerce: '', email: '', phone: '', iban: '',
      address: { street: '', house_number: '', postal_code: '', city: '', country: '' }, updated_at: '',
    })

    expect(await screen.findByRole('region', { name: 'Seller' })).toHaveTextContent('Cantinho B.V.')
    await waitFor(() => expect(window.print).toHaveBeenCalledTimes(1))
  })
})
