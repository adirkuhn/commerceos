// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../api', () => ({
  lookupAddress: vi.fn(),
  getDeliveryPriceEstimate: vi.fn(),
}))

import { AddressConfirmModal, splitStreetAndNumber, addressToRecipientDraft } from '../AddressConfirmModal'
import * as shippingApi from '../api'
import type { Address } from '../../oms/types'

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(shippingApi.lookupAddress).mockResolvedValue(null)
  vi.mocked(shippingApi.getDeliveryPriceEstimate).mockResolvedValue([])
})

describe('splitStreetAndNumber', () => {
  it('splits a trailing number', () => {
    expect(splitStreetAndNumber('Damrak 1')).toEqual({ street: 'Damrak', number: '1', suffix: '' })
  })

  it('splits a trailing number with a letter suffix (complement)', () => {
    expect(splitStreetAndNumber('Damrak 1A')).toEqual({ street: 'Damrak', number: '1', suffix: 'A' })
  })

  it('falls back to the whole string as street when there is no trailing number and there is no address2', () => {
    expect(splitStreetAndNumber('Damrak')).toEqual({ street: 'Damrak', number: '', suffix: '' })
  })

  // Regression: some NL WooCommerce checkouts/address plugins split street
  // and house number into separate address lines (address_1: "Fossa
  // Iberica", address_2: "9") instead of one combined string -- real order
  // data, house number silently went missing before this fallback existed.
  it('falls back to address2 for the house number when address1 has none and address2 is just a number', () => {
    expect(splitStreetAndNumber('Fossa Iberica', '9')).toEqual({ street: 'Fossa Iberica', number: '9', suffix: '' })
  })

  it('falls back to address2 including a letter suffix', () => {
    expect(splitStreetAndNumber('Fossa Iberica', '9A')).toEqual({ street: 'Fossa Iberica', number: '9', suffix: 'A' })
  })

  it('ignores address2 when it is not just a number (e.g. an apartment/unit line)', () => {
    expect(splitStreetAndNumber('Fossa Iberica', 'Apartment 3')).toEqual({ street: 'Fossa Iberica', number: '', suffix: '' })
  })

  it('prefers a number already found in address1 over address2', () => {
    expect(splitStreetAndNumber('Damrak 1', '9')).toEqual({ street: 'Damrak', number: '1', suffix: '' })
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

  it('falls back to address_2 for the house number when address_1 has none', () => {
    const address: Address = {
      first_name: 'Letícia', last_name: 'Omena', address_1: 'Fossa Iberica', address_2: '9',
      city: 'Houten', postcode: '3995 XH', country: 'NL',
    }
    const draft = addressToRecipientDraft(address)
    expect(draft.street).toBe('Fossa Iberica')
    expect(draft.number).toBe('9')
  })

  it('falls back to the order billing email when the shipping address has none', () => {
    const address: Address = { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' }
    expect(addressToRecipientDraft(address, 'billing@example.com').email).toBe('billing@example.com')
  })

  it('prefers the shipping address\'s own email over the billing fallback', () => {
    const address: Address = { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL', email: 'shipping@example.com' }
    expect(addressToRecipientDraft(address, 'billing@example.com').email).toBe('shipping@example.com')
  })
})

describe('AddressConfirmModal', () => {
  const address: Address = { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1A', city: 'Amsterdam', postcode: '1012AB', country: 'NL', email: 'jane@example.com' }

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

    // Submitting now runs the address check first (if it hasn't already) --
    // this order's postal code has no lookup mock set up so it resolves
    // not_found, which must not block the submit.
    await waitFor(() => expect(onNext).toHaveBeenCalledWith({
      carrier: 1,
      recipient: expect.objectContaining({ person: 'Jane Doe', street: 'Damrak', number: '2', city: 'Amsterdam' }),
    }))
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

  it('prefills email from fallbackEmail (order billing email) when the shipping address has none', () => {
    const addressNoEmail: Address = { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1A', city: 'Amsterdam', postcode: '1012AB', country: 'NL' }
    render(<AddressConfirmModal token="test-token" initialAddress={addressNoEmail} fallbackEmail="billing@example.com" onNext={vi.fn()} onClose={vi.fn()} />)

    expect(screen.getByLabelText('Email')).toHaveValue('billing@example.com')
  })
})

describe('AddressConfirmModal — address check (PDOK lookup)', () => {
  const nlAddress: Address = { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL', email: 'jane@example.com' }

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

describe('AddressConfirmModal — submit-time validation gate', () => {
  const nlAddress: Address = { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL', email: 'jane@example.com' }

  it('runs the check on submit if it never ran before (prefilled fields staff never touched), and blocks on a mismatch', async () => {
    vi.mocked(shippingApi.lookupAddress).mockResolvedValue({ street: 'Damstraat', city: 'Amsterdam' })
    const onNext = vi.fn()
    render(<AddressConfirmModal token="test-token" initialAddress={nlAddress} onNext={onNext} onClose={vi.fn()} />)

    await userEvent.click(screen.getByRole('button', { name: 'Review label' }))

    expect(await screen.findByText(/Possible typo/)).toBeInTheDocument()
    expect(onNext).not.toHaveBeenCalled()
  })

  it('proceeds after "Use suggested" resolves the mismatch', async () => {
    vi.mocked(shippingApi.lookupAddress).mockResolvedValue({ street: 'Damstraat', city: 'Amsterdam' })
    const onNext = vi.fn()
    render(<AddressConfirmModal token="test-token" initialAddress={nlAddress} onNext={onNext} onClose={vi.fn()} />)

    await userEvent.click(screen.getByRole('button', { name: 'Review label' }))
    await screen.findByText(/Possible typo/)
    await userEvent.click(screen.getByRole('button', { name: 'Use suggested' }))
    await userEvent.click(screen.getByRole('button', { name: 'Review label' }))

    expect(onNext).toHaveBeenCalledWith({
      carrier: 1,
      recipient: expect.objectContaining({ street: 'Damstraat', city: 'Amsterdam' }),
    })
  })

  it('proceeds after "Keep mine" dismisses the mismatch, without re-blocking', async () => {
    vi.mocked(shippingApi.lookupAddress).mockResolvedValue({ street: 'Damstraat', city: 'Amsterdam' })
    const onNext = vi.fn()
    render(<AddressConfirmModal token="test-token" initialAddress={nlAddress} onNext={onNext} onClose={vi.fn()} />)

    await userEvent.click(screen.getByRole('button', { name: 'Review label' }))
    await screen.findByText(/Possible typo/)
    await userEvent.click(screen.getByRole('button', { name: 'Keep mine' }))
    await userEvent.click(screen.getByRole('button', { name: 'Review label' }))

    expect(onNext).toHaveBeenCalledWith({
      carrier: 1,
      recipient: expect.objectContaining({ street: 'Damrak', city: 'Amsterdam' }),
    })
  })

  it('a lookup failure at submit time does not block -- the API being unavailable is never a reason to stop staff', async () => {
    vi.mocked(shippingApi.lookupAddress).mockRejectedValue(new Error('network error'))
    const onNext = vi.fn()
    render(<AddressConfirmModal token="test-token" initialAddress={nlAddress} onNext={onNext} onClose={vi.fn()} />)

    await userEvent.click(screen.getByRole('button', { name: 'Review label' }))

    await waitFor(() => expect(onNext).toHaveBeenCalled())
  })

  it('editing a field after a match/dismissal re-arms the gate for the next submit', async () => {
    vi.mocked(shippingApi.lookupAddress).mockResolvedValue({ street: 'Damstraat', city: 'Amsterdam' })
    const onNext = vi.fn()
    render(<AddressConfirmModal token="test-token" initialAddress={nlAddress} onNext={onNext} onClose={vi.fn()} />)

    await userEvent.click(screen.getByRole('button', { name: 'Review label' }))
    await screen.findByText(/Possible typo/)
    await userEvent.click(screen.getByRole('button', { name: 'Keep mine' }))

    // Staff edits the street after dismissing -- the stale "dismissed"
    // outcome must not silently wave through a brand new change unchecked.
    await userEvent.clear(screen.getByLabelText('Street'))
    await userEvent.type(screen.getByLabelText('Street'), 'Something Else')
    await userEvent.click(screen.getByRole('button', { name: 'Review label' }))

    expect(await screen.findByText(/Possible typo/)).toBeInTheDocument()
    expect(onNext).not.toHaveBeenCalled()
  })
})

describe('AddressConfirmModal — estimated label cost', () => {
  const nlAddress: Address = { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL', email: 'jane@example.com' }

  it('shows the cheapest estimated cost once postal code + number are known for an NL address', async () => {
    vi.mocked(shippingApi.getDeliveryPriceEstimate).mockResolvedValue([
      { price_comment: 'morning', amount_minor_units: 1500, currency: 'EUR' },
      { price_comment: 'standard', amount_minor_units: 1000, currency: 'EUR' },
    ])
    render(<AddressConfirmModal token="test-token" connectionId="conn-1" initialAddress={nlAddress} defaultCarrier={1} onNext={vi.fn()} onClose={vi.fn()} />)

    await waitFor(() => expect(shippingApi.getDeliveryPriceEstimate).toHaveBeenCalledWith('test-token', {
      connection_id: 'conn-1', cc: 'NL', postal_code: '1012AB', number: '1', carrier: 1,
    }))
    expect(await screen.findByText(/standard/)).toBeInTheDocument()
    expect(screen.getByText(/€\s?10[.,]00/)).toBeInTheDocument()
    expect(screen.queryByText(/morning/)).not.toBeInTheDocument()
  })

  it('does not fetch an estimate without a connectionId', async () => {
    render(<AddressConfirmModal token="test-token" initialAddress={nlAddress} defaultCarrier={1} onNext={vi.fn()} onClose={vi.fn()} />)

    await userEvent.click(screen.getByLabelText('Postal code'))
    await userEvent.tab()

    expect(shippingApi.getDeliveryPriceEstimate).not.toHaveBeenCalled()
  })

  it('does not fetch an estimate for a country other than NL/BE', async () => {
    render(<AddressConfirmModal token="test-token" connectionId="conn-1" initialAddress={{ ...nlAddress, country: 'DE' }} defaultCarrier={1} onNext={vi.fn()} onClose={vi.fn()} />)

    await userEvent.click(screen.getByLabelText('Street'))
    await userEvent.tab()

    expect(shippingApi.getDeliveryPriceEstimate).not.toHaveBeenCalled()
  })

  it('re-fetches when the carrier changes', async () => {
    vi.mocked(shippingApi.getDeliveryPriceEstimate).mockResolvedValue([{ price_comment: 'standard', amount_minor_units: 1000, currency: 'EUR' }])
    render(<AddressConfirmModal token="test-token" connectionId="conn-1" initialAddress={nlAddress} defaultCarrier={1} onNext={vi.fn()} onClose={vi.fn()} />)
    await waitFor(() => expect(shippingApi.getDeliveryPriceEstimate).toHaveBeenCalledTimes(1))

    await userEvent.selectOptions(screen.getByLabelText('Carrier'), '4')

    await waitFor(() => expect(shippingApi.getDeliveryPriceEstimate).toHaveBeenLastCalledWith('test-token', expect.objectContaining({ carrier: 4 })))
  })

  it('silently shows nothing if the estimate call fails', async () => {
    vi.mocked(shippingApi.getDeliveryPriceEstimate).mockRejectedValue(new Error('network error'))
    render(<AddressConfirmModal token="test-token" connectionId="conn-1" initialAddress={nlAddress} defaultCarrier={1} onNext={vi.fn()} onClose={vi.fn()} />)

    await waitFor(() => expect(shippingApi.getDeliveryPriceEstimate).toHaveBeenCalled())
    expect(screen.queryByText(/Estimated cost/)).not.toBeInTheDocument()
  })
})
