// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { ShippingProviderPicker } from '../ShippingProviderPicker'
import { shippingConnectionsOf } from '../../test/shippingConnections'

const { connections: TWO } = shippingConnectionsOf([{ id: 'mp', provider: 'myparcel' }, { id: 'pn', provider: 'postnl' }])

describe('ShippingProviderPicker', () => {
  it('shows nothing without connections', () => {
    const { container } = render(<ShippingProviderPicker connections={[]} selectedId={undefined} onSelect={vi.fn()} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('names the only provider instead of a dropdown', () => {
    render(<ShippingProviderPicker connections={TWO.slice(0, 1)} selectedId="mp" onSelect={vi.fn()} />)
    expect(screen.getByText('Shipping via MyParcel')).toBeInTheDocument()
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
  })

  it('offers every connection, preselected, and reports a switch', async () => {
    const onSelect = vi.fn()
    render(<ShippingProviderPicker connections={TWO} selectedId="pn" onSelect={onSelect} />)
    const select = screen.getByLabelText('Shipping provider')
    expect(select).toHaveValue('pn')
    expect(screen.getAllByRole('option').map(o => o.textContent)).toEqual(['MyParcel', 'PostNL'])
    await userEvent.selectOptions(select, 'mp')
    expect(onSelect).toHaveBeenCalledWith('mp')
  })
})
