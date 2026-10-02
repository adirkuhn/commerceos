// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../api', () => ({
  backfillWooOrderPayments: vi.fn(),
}))

import * as api from '../api'
import { BackfillOrderPaymentsCard } from '../BackfillOrderPaymentsCard'

beforeEach(() => vi.clearAllMocks())

describe('BackfillOrderPaymentsCard', () => {
  it('runs the backfill and reports how many orders were checked and updated', async () => {
    vi.mocked(api.backfillWooOrderPayments)
      .mockResolvedValueOnce({ checked: 25, updated: 20, errors: [], next_cursor: 'c1' })
      .mockResolvedValueOnce({ checked: 3, updated: 1, errors: ['woo order 9: fetch: not found'], next_cursor: null })
    render(<BackfillOrderPaymentsCard token="tok" connectionId="conn-1" />)

    await userEvent.click(screen.getByRole('button', { name: 'Fetch missing payment info' }))

    expect(await screen.findByText('Done: 28 orders checked, 21 updated.')).toBeInTheDocument()
    expect(screen.getByText('Some orders could not be checked:')).toBeInTheDocument()
    expect(screen.getByText('woo order 9: fetch: not found')).toBeInTheDocument()
  })

  it('says when nothing was missing', async () => {
    vi.mocked(api.backfillWooOrderPayments).mockResolvedValueOnce({ checked: 0, updated: 0, errors: [], next_cursor: null })
    render(<BackfillOrderPaymentsCard token="tok" connectionId="conn-1" />)

    await userEvent.click(screen.getByRole('button', { name: 'Fetch missing payment info' }))

    expect(await screen.findByText('Every paid order already has its payment info.')).toBeInTheDocument()
  })

  it('shows a failure', async () => {
    vi.mocked(api.backfillWooOrderPayments).mockRejectedValueOnce(new Error('woocommerce request failed'))
    render(<BackfillOrderPaymentsCard token="tok" connectionId="conn-1" />)

    await userEvent.click(screen.getByRole('button', { name: 'Fetch missing payment info' }))

    expect(await screen.findByText('woocommerce request failed')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Fetch missing payment info' })).toBeEnabled()
  })
})
