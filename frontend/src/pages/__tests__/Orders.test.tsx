// @vitest-environment jsdom
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { act } from 'react'
import { describe, it, expect } from 'vitest'
import Orders from '../Orders'
import { renderAt } from '../../test/renderAt'

// Orders is still backed by hardcoded demo data (ORDERS in Orders.tsx);
// these tests cover only that its view state lives in the URL.
describe('Orders URLs (refresh + Back)', () => {
  it('clicking an order pushes /orders/:id and opens its panel; Back closes it', async () => {
    const { router } = renderAt(<Orders />, '/orders')
    await userEvent.click(await screen.findByText('ORD-10042'))
    expect(router.state.location.pathname).toBe('/orders/ORD-10042')
    expect(await screen.findByRole('heading', { name: 'ORD-10042' })).toBeInTheDocument()

    await act(() => router.navigate(-1))
    expect(router.state.location.pathname).toBe('/orders')
    expect(screen.queryByRole('heading', { name: 'ORD-10042' })).not.toBeInTheDocument()
  })

  it('/orders/:id opens that order on load', async () => {
    renderAt(<Orders />, '/orders', '/orders/ORD-10040')
    expect(await screen.findByRole('heading', { name: 'ORD-10040' })).toBeInTheDocument()
  })

  it('an unknown order id shows no panel rather than crashing', async () => {
    renderAt(<Orders />, '/orders', '/orders/nope')
    expect(await screen.findByText('ORD-10042')).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'nope' })).not.toBeInTheDocument()
  })

  it('status tab lives in ?status=', async () => {
    const { router } = renderAt(<Orders />, '/orders', '/orders?status=picking')
    const table = await screen.findByRole('table')
    expect(within(table).getByText('ORD-10040')).toBeInTheDocument()
    expect(within(table).queryByText('ORD-10042')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: /^Shipped/ }))
    expect(router.state.location.search).toBe('?status=shipped')
  })

  it('search lives in ?q=', async () => {
    const { router } = renderAt(<Orders />, '/orders', '/orders?q=rossi')
    const table = await screen.findByRole('table')
    expect(within(table).getByText('ORD-10039')).toBeInTheDocument()
    expect(within(table).queryByText('ORD-10042')).not.toBeInTheDocument()
    await userEvent.type(screen.getByPlaceholderText('Search order ID, customer...'), 'x')
    expect(router.state.location.search).toBe('?q=rossix')
  })
})
