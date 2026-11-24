// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { ShippingAddressesCard } from '../ShippingAddressesCard'
import type { ShippingAddress } from '../../types'

const main: ShippingAddress = {
  id: 'sa-1', customer_id: 'c-1', label: 'Main', first_name: '', last_name: '', company: '', address_1: 'Main 1', address_2: '',
  city: 'Utrecht', state: '', postcode: '3511', country: 'NL', phone: '', is_default: true,
}
const dock: ShippingAddress = { ...main, id: 'sa-2', label: 'Dock', address_1: 'Dock 2', is_default: false }

function renderCard(over: Partial<React.ComponentProps<typeof ShippingAddressesCard>> = {}) {
  const props = {
    addresses: [main, dock], error: null,
    onAdd: vi.fn().mockResolvedValue(undefined), onUpdate: vi.fn().mockResolvedValue(undefined),
    onRemove: vi.fn().mockResolvedValue(undefined), onMakeDefault: vi.fn().mockResolvedValue(undefined), ...over,
  }
  render(<ShippingAddressesCard {...props} />)
  return props
}

describe('ShippingAddressesCard', () => {
  it('lists each address with its label, line and the default badge', () => {
    renderCard()
    const [first, second] = screen.getAllByRole('listitem')
    expect(first).toHaveTextContent('Main')
    expect(first).toHaveTextContent('Main 1, 3511 Utrecht, NL')
    expect(within(first).getByText('Default')).toBeInTheDocument()
    expect(within(second).queryByText('Default')).not.toBeInTheDocument()
  })

  it('makes another address the default', async () => {
    const { onMakeDefault } = renderCard()
    expect(screen.queryByRole('button', { name: 'Make Main the default' })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Make Dock the default' }))
    await waitFor(() => expect(onMakeDefault).toHaveBeenCalledWith('sa-2'))
  })

  it('adds an address; an address line is needed', async () => {
    const { onAdd } = renderCard({ addresses: [] })
    expect(screen.getByText('No shipping addresses yet.')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Add shipping address' }))
    const save = screen.getByRole('button', { name: 'Save' })
    expect(save).toBeDisabled()
    fireEvent.change(screen.getByLabelText('Label'), { target: { value: 'Warehouse' } })
    fireEvent.change(screen.getByLabelText('Address line 1'), { target: { value: 'Dock 2' } })
    fireEvent.change(screen.getByLabelText('City'), { target: { value: 'Utrecht' } })
    fireEvent.click(screen.getByRole('checkbox', { name: 'Make it the default' }))
    fireEvent.click(save)

    await waitFor(() => expect(onAdd).toHaveBeenCalledWith(expect.objectContaining({
      label: 'Warehouse', address_1: 'Dock 2', city: 'Utrecht', is_default: true,
    })))
    await waitFor(() => expect(screen.queryByLabelText('Label')).not.toBeInTheDocument())
  })

  it('edits an address', async () => {
    const { onUpdate } = renderCard()
    fireEvent.click(screen.getByRole('button', { name: 'Edit Dock' }))
    expect(screen.getByLabelText('Address line 1')).toHaveValue('Dock 2')
    expect(screen.queryByRole('checkbox', { name: 'Make it the default' })).not.toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('City'), { target: { value: 'Delft' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(onUpdate).toHaveBeenCalledWith('sa-2', expect.objectContaining({ address_1: 'Dock 2', city: 'Delft' })))
  })

  it('removes an address after confirming', async () => {
    const { onRemove } = renderCard()
    fireEvent.click(screen.getByRole('button', { name: 'Edit Dock' }))
    fireEvent.click(screen.getByRole('button', { name: 'Remove' }))
    fireEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: 'Remove' }))

    await waitFor(() => expect(onRemove).toHaveBeenCalledWith('sa-2'))
  })

  it('shows why a save failed', async () => {
    renderCard({ addresses: [], onAdd: vi.fn().mockRejectedValue(new Error('boom')) })
    fireEvent.click(screen.getByRole('button', { name: 'Add shipping address' }))
    fireEvent.change(screen.getByLabelText('Address line 1'), { target: { value: 'Dock 2' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })
})
