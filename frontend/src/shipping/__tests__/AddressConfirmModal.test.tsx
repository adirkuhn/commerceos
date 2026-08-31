// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../api', () => ({
  lookupAddress: vi.fn(),
}))

import { AddressConfirmModal, splitStreetAndNumber, addressToRecipientDraft } from '../AddressConfirmModal'
import * as shippingApi from '../api'
import type { Address } from '../../oms/types'

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(shippingApi.lookupAddress).mockResolvedValue(null)
})

describe('splitStreetAndNumber', () => {
  it('splits a trailing number', () => {
    expect(splitStreetAndNumber('Damrak 1')).toEqual({ street: 'Damrak', number: '1', suffix: '' })
  })

  it('splits a trailing number with a letter suffix (complement)', () => {
    expect(splitStreetAndNumber('Damrak 1A')).toEqual({ street: 'Damrak', number: '1', suffix: 'A' })
  })

  it('falls back to the whole string as street when there is no trailing number', () => {
    expect(splitStreetAndNumber('Damrak')).toEqual({ street: 'Damrak', number: '', suffix: '' })
  })
})

describe('addressToRecipientDraft', () => {
  it('maps an oms Address into a MyParcel-shaped recipient draft', () => {
    const address: Address = {
      first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1A',
      city: 'Amsterdam', postcode: '1012AB', country: 'nl', phone: '0600000000',
    }
    expect(addressToRecipientDraft(address)).toEqual({
      person: 'Jane Doe', company: '', street: 'Damrak', number: '1', number_suffix: 'A',
      postal_code: '1012AB', city: 'Amsterdam', cc: 'NL', email: '', phone: '0600000000',
    })
  })

  it('handles an undefined address (manual order, no shipping address on file)', () => {
    expect(addressToRecipientDraft(undefined)).toEqual({
      person: '', company: '', street: '', number: '', number_suffix: '',
      postal_code: '', city: '', cc: '', email: '', phone: '',
    })
  })
})

describe('AddressConfirmModal', () => {
  const address: Address = { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1A', city: 'Amsterdam', postcode: '1012AB', country: 'NL' }

  it('prefills the form from the order address and hands off the edited/confirmed recipient + carrier', async () => {
    const onNext = vi.fn()
    render(<AddressConfirmModal token="test-token" initialAddress={address} defaultCarrier={1} onNext={onNext} onClose={vi.fn()} />)

    expect(screen.getByLabelText('Name')).toHaveValue('Jane Doe')
    expect(screen.getByLabelText('Street')).toHaveValue('Damrak')
    expect(screen.getByLabelText('Number')).toHaveValue('1')
    expect(screen.getByLabelText('Complement (toevoeging)')).toHaveValue('A')

    // Staff correct the number before confirming -- proves edits aren't
    // silently overwritten/ignored.
    await userEvent.clear(screen.getByLabelText('Number'))
    await userEvent.type(screen.getByLabelText('Number'), '2')
    await userEvent.click(screen.getByRole('button', { name: 'Review label' }))

    expect(onNext).toHaveBeenCalledWith({
      carrier: 1,
      recipient: expect.objectContaining({ person: 'Jane Doe', street: 'Damrak', number: '2', city: 'Amsterdam' }),
    })
  })

  // Reaching this modal via LabelPreviewModal's "Back to edit" must not
  // reset staff's edits back to the order's raw address -- initialRecipient
  // wins over re-deriving from initialAddress.
  it('prefers initialRecipient/initialCarrier over the order address when returning from the preview step', () => {
    const onNext = vi.fn()
    render(
      <AddressConfirmModal
        token="test-token"
        initialAddress={address}
        initialRecipient={{ person: 'Edited Name', street: 'Edited Street', city: 'Rotterdam', cc: 'NL' }}
        initialCarrier={4}
        onNext={onNext}
        onClose={vi.fn()}
      />,
    )

    expect(screen.getByLabelText('Name')).toHaveValue('Edited Name')
    expect(screen.getByLabelText('Street')).toHaveValue('Edited Street')
    expect(screen.getByLabelText<HTMLSelectElement>('Carrier').value).toBe('4')
  })

  it('calls onClose when the close button is clicked', async () => {
    const onClose = vi.fn()
    render(<AddressConfirmModal token="test-token" initialAddress={address} onNext={vi.fn()} onClose={onClose} />)

    await userEvent.click(screen.getByLabelText('Close'))

    expect(onClose).toHaveBeenCalled()
  })
})

describe('AddressConfirmModal — address check (PDOK lookup)', () => {
  const nlAddress: Address = { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' }

  it('fills blank street/city from the lookup result once postal code and number are both entered', async () => {
    vi.mocked(shippingApi.lookupAddress).mockResolvedValue({ street: 'Damrak', city: 'Amsterdam' })
    render(<AddressConfirmModal token="test-token" onNext={vi.fn()} onClose={vi.fn()} />)

    await userEvent.type(screen.getByLabelText('Country code'), 'NL')
    await userEvent.type(screen.getByLabelText('Postal code'), '1012AB')
    await userEvent.type(screen.getByLabelText('Number'), '1')
    await userEvent.tab()

    await waitFor(() => expect(screen.getByLabelText('Street')).toHaveValue('Damrak'))
    expect(screen.getByLabelText('City')).toHaveValue('Amsterdam')
    expect(shippingApi.lookupAddress).toHaveBeenCalledWith('test-token', '1012AB', '1')
  })

  it('flags a mismatch between what staff typed and what the registry says, with a one-click fix', async () => {
    vi.mocked(shippingApi.lookupAddress).mockResolvedValue({ street: 'Damstraat', city: 'Amsterdam' })
    render(<AddressConfirmModal token="test-token" initialAddress={nlAddress} onNext={vi.fn()} onClose={vi.fn()} />)

    await userEvent.click(screen.getByLabelText('Postal code'))
    await userEvent.tab()

    expect(await screen.findByText(/Possible typo/)).toBeInTheDocument()
    expect(screen.getByText(/Damrak, Amsterdam/)).toBeInTheDocument()
    expect(screen.getByText(/Damstraat, Amsterdam/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Use suggested' }))

    expect(screen.getByLabelText('Street')).toHaveValue('Damstraat')
    expect(screen.queryByText(/Possible typo/)).not.toBeInTheDocument()
  })

  it('"Keep mine" dismisses the mismatch banner without changing the fields', async () => {
    vi.mocked(shippingApi.lookupAddress).mockResolvedValue({ street: 'Damstraat', city: 'Amsterdam' })
    render(<AddressConfirmModal token="test-token" initialAddress={nlAddress} onNext={vi.fn()} onClose={vi.fn()} />)

    await userEvent.click(screen.getByLabelText('Postal code'))
    await userEvent.tab()
    await screen.findByText(/Possible typo/)

    await userEvent.click(screen.getByRole('button', { name: 'Keep mine' }))

    expect(screen.queryByText(/Possible typo/)).not.toBeInTheDocument()
    expect(screen.getByLabelText('Street')).toHaveValue('Damrak')
  })

  it('shows a soft not-found note when nothing matches, without blocking the form', async () => {
    vi.mocked(shippingApi.lookupAddress).mockResolvedValue(null)
    render(<AddressConfirmModal token="test-token" initialAddress={nlAddress} onNext={vi.fn()} onClose={vi.fn()} />)

    await userEvent.click(screen.getByLabelText('Postal code'))
    await userEvent.tab()

    expect(await screen.findByText(/No match found/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Review label' })).not.toBeDisabled()
  })

  it('does not run the check for a non-NL address', async () => {
    render(<AddressConfirmModal token="test-token" initialAddress={{ ...nlAddress, country: 'DE' }} onNext={vi.fn()} onClose={vi.fn()} />)

    await userEvent.click(screen.getByLabelText('Postal code'))
    await userEvent.tab()

    expect(shippingApi.lookupAddress).not.toHaveBeenCalled()
  })
})
