// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { SoldExpiryNote } from '../SoldExpiryNote'
import { DateFormatProvider } from '../../format/DateFormatProvider'

vi.mock('../../settings/api', () => ({
  getTenantSettings: vi.fn().mockResolvedValue({ date_format: 'YYYY-MM-DD', timezone: '' }),
}))

describe('SoldExpiryNote', () => {
  it('renders the sold expiry date in the default date format', () => {
    render(<SoldExpiryNote date="2027-03-09" />)
    expect(screen.getByText(/Sold exp\./)).toHaveTextContent('Sold exp. 09/03/2027')
  })

  it('explains that the date is what WooCommerce reported, not the allocated batch', () => {
    render(<SoldExpiryNote date="2027-03-09" />)
    expect(screen.getByText(/Sold exp\./)).toHaveAttribute('title', expect.stringMatching(/WooCommerce/))
  })

  it('renders nothing when the line has no sold expiry date', () => {
    const { container } = render(<SoldExpiryNote date={null} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders nothing when the date is undefined', () => {
    const { container } = render(<SoldExpiryNote date={undefined} />)
    expect(container).toBeEmptyDOMElement()
  })

  it("uses the tenant's configured date format", async () => {
    render(<DateFormatProvider token="t"><SoldExpiryNote date="2027-03-09" /></DateFormatProvider>)
    expect(await screen.findByText('Sold exp. 2027-03-09')).toBeInTheDocument()
  })

  it('applies the caller-supplied className so each surface keeps its own styling', () => {
    render(<SoldExpiryNote date="2027-03-09" className="text-xs text-gray-400" />)
    expect(screen.getByText(/Sold exp\./)).toHaveClass('text-xs', 'text-gray-400')
  })
})
