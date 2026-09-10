// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { AddressForm } from '../AddressForm'

describe('AddressForm', () => {
  it('submits the entered values', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(<AddressForm onSubmit={onSubmit} />)

    await user.type(screen.getByLabelText('First name'), 'Morgan')
    await user.type(screen.getByLabelText('Last name'), 'Lee')
    await user.type(screen.getByLabelText('Street address'), '482 Ridgeview Ave')
    await user.type(screen.getByLabelText('City'), 'Boulder')
    await user.type(screen.getByLabelText('Postal code'), '80302')
    await user.click(screen.getByRole('button', { name: 'Save address' }))

    expect(onSubmit).toHaveBeenCalledTimes(1)
    expect(onSubmit.mock.calls[0][0]).toEqual(expect.objectContaining({
      firstName: 'Morgan', lastName: 'Lee', line1: '482 Ridgeview Ave', city: 'Boulder', postalCode: '80302', country: 'US',
    }))
  })

  it('shows validation errors and does not submit when required fields are empty', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(<AddressForm onSubmit={onSubmit} />)

    await user.click(screen.getByRole('button', { name: 'Save address' }))

    expect(await screen.findByText('First name is required')).toBeInTheDocument()
    expect(await screen.findByText('Street address is required')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('shows the VAT field only when showVatField is set', () => {
    const { rerender } = render(<AddressForm onSubmit={vi.fn()} />)
    expect(screen.queryByLabelText(/VAT/)).not.toBeInTheDocument()

    rerender(<AddressForm onSubmit={vi.fn()} showVatField />)
    expect(screen.getByLabelText(/VAT/)).toBeInTheDocument()
  })

  it('prefills from defaultValues', () => {
    render(<AddressForm onSubmit={vi.fn()} defaultValues={{ firstName: 'Alex', city: 'Denver' }} />)
    expect(screen.getByLabelText('First name')).toHaveValue('Alex')
    expect(screen.getByLabelText('City')).toHaveValue('Denver')
  })
})
