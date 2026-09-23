// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { PackingOrderQueue } from '../PackingOrderQueue'
import type { SalesOrder } from '../../oms/types'

function order(overrides: Partial<SalesOrder>): SalesOrder {
  return {
    id: 'so-1', tenant_id: 't', source: 'woocommerce', external_order_id: 'ext-1', number: '1001', order_key: '',
    status: 'processing', currency_code: 'EUR', warehouse_id: 'wh-1', billing: { first_name: 'Ann', last_name: 'Lee' },
    total_minor_units: 1250, fulfilled: true, packed: false,
    ...overrides,
  } as SalesOrder
}

const orders = [
  order({ id: 'so-1', number: '1001', fulfilled: true }),
  order({ id: 'so-2', number: '1002', fulfilled: false, billing: { first_name: 'Bo', last_name: 'Ng' } }),
]

function renderQueue(props: Partial<Parameters<typeof PackingOrderQueue>[0]> = {}) {
  const onSelect = vi.fn()
  const onOpenPickList = vi.fn()
  render(
    <PackingOrderQueue
      orders={orders} loading={false} error="" activeOrderId={null} activeInProgress={false}
      onSelect={onSelect} onOpenPickList={onOpenPickList} {...props}
    />,
  )
  return { onSelect, onOpenPickList }
}

describe('PackingOrderQueue', () => {
  it('marks orders that skipped picking', () => {
    renderQueue()

    expect(screen.getAllByText('Not picked')).toHaveLength(1)
    expect(screen.getByRole('button', { name: /#1002/ })).toHaveTextContent('Not picked')
  })

  it('selecting an order row calls onSelect', async () => {
    const user = userEvent.setup()
    const { onSelect } = renderQueue()

    await user.click(screen.getByRole('button', { name: /#1001/ }))

    expect(onSelect).toHaveBeenCalledWith(orders[0])
  })

  it('pick list button is disabled until orders are ticked, then opens with the ticked ids', async () => {
    const user = userEvent.setup()
    const { onOpenPickList, onSelect } = renderQueue()
    const pickListButton = screen.getByRole('button', { name: /Pick list/ })
    expect(pickListButton).toBeDisabled()

    await user.click(screen.getByRole('checkbox', { name: 'Add #1001 to pick list' }))
    await user.click(screen.getByRole('checkbox', { name: 'Add #1002 to pick list' }))
    expect(pickListButton).toHaveTextContent('Pick list (2)')
    await user.click(pickListButton)

    expect(onOpenPickList).toHaveBeenCalledWith(['so-1', 'so-2'])
    expect(onSelect).not.toHaveBeenCalled()
  })

  it('select all ticks every visible order', async () => {
    const user = userEvent.setup()
    const { onOpenPickList } = renderQueue()

    await user.click(screen.getByRole('checkbox', { name: 'Select all orders' }))
    await user.click(screen.getByRole('button', { name: /Pick list/ }))

    expect(onOpenPickList).toHaveBeenCalledWith(['so-1', 'so-2'])
  })

  it('search narrows the list', async () => {
    const user = userEvent.setup()
    renderQueue()

    await user.type(screen.getByPlaceholderText('Search orders...'), 'bo ng')

    expect(screen.queryByRole('button', { name: /#1001/ })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /#1002/ })).toBeInTheDocument()
  })

  it('shows loading, error and empty states', () => {
    const { rerender } = render(
      <PackingOrderQueue orders={[]} loading error="" activeOrderId={null} activeInProgress={false} onSelect={vi.fn()} onOpenPickList={vi.fn()} />,
    )
    expect(screen.getByText('Loading…')).toBeInTheDocument()

    rerender(<PackingOrderQueue orders={[]} loading={false} error="boom" activeOrderId={null} activeInProgress={false} onSelect={vi.fn()} onOpenPickList={vi.fn()} />)
    expect(screen.getByText('boom')).toBeInTheDocument()

    rerender(<PackingOrderQueue orders={[]} loading={false} error="" activeOrderId={null} activeInProgress={false} onSelect={vi.fn()} onOpenPickList={vi.fn()} />)
    expect(screen.getByText('No orders ready to pack')).toBeInTheDocument()
  })
})
