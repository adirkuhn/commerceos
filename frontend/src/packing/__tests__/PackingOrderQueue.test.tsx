// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { PackingOrderQueue } from '../PackingOrderQueue'
import { AmountVisibilityProvider } from '../../format/amountVisibility'
import type { QueueEntry, SalesOrder } from '../../oms/types'

function order(overrides: Partial<SalesOrder>): SalesOrder {
  return {
    id: 'so-1', tenant_id: 't', source: 'woocommerce', external_order_id: 'ext-1', number: '1001', order_key: '',
    status: 'processing', currency_code: 'EUR', warehouse_id: 'wh-1', billing: { first_name: 'Ann', last_name: 'Lee' },
    total_minor_units: 1250, fulfilled: true, packed: false,
    ...overrides,
  } as SalesOrder
}

const twelveMinAgo = () => new Date(Date.now() - 12 * 60_000).toISOString()

const entries: QueueEntry[] = [
  { sales_order: order({ id: 'so-1', number: '1001', fulfilled: true, shipping_method_title: 'PostNL pakket' }), lock: null, in_packing: false },
  { sales_order: order({ id: 'so-2', number: '1002', fulfilled: false, billing: { first_name: 'Bo', last_name: 'Ng' } }), lock: null, in_packing: false },
  {
    sales_order: order({ id: 'so-3', number: '1003', fulfilled: false }),
    lock: { activity: 'picking', user_id: 'picker-1', since: twelveMinAgo() }, in_packing: false,
  },
  {
    sales_order: order({ id: 'so-4', number: '1004' }),
    lock: { activity: 'packing', user_id: 'me', since: twelveMinAgo() }, in_packing: false,
  },
]

const nameFor = (id: string) => (id === 'picker-1' ? 'Carl Picker' : id)

type Props = Parameters<typeof PackingOrderQueue>[0]

function renderQueue(props: Partial<Props> = {}) {
  const onSelect = vi.fn()
  const onOpenPickList = vi.fn()
  render(
    <PackingOrderQueue
      entries={entries} loading={false} error="" activeOrderId={null} activeInProgress={false}
      currentUserId="me" isAdmin={false} nameFor={nameFor}
      onSelect={onSelect} onOpenPickList={onOpenPickList} {...props}
    />,
  )
  return { onSelect, onOpenPickList }
}

function row(number: string): HTMLElement {
  return screen.getByText(`#${number}`).closest('li') as HTMLElement
}

describe('PackingOrderQueue', () => {
  it('shows each order\'s shipping method, and nothing when it has none', () => {
    renderQueue()

    expect(within(row('1001')).getByTitle('Shipping method')).toHaveTextContent('PostNL pakket')
    expect(within(row('1002')).queryByTitle('Shipping method')).not.toBeInTheDocument()
  })

  it('lists every order, marking ones that skipped picking', () => {
    renderQueue()

    expect(screen.getByText('4 to pack')).toBeInTheDocument()
    expect(within(row('1002')).getByText('Not picked')).toBeInTheDocument()
    expect(within(row('1001')).queryByText('Not picked')).not.toBeInTheDocument()
  })

  it('an order someone else holds is read-only and says who and for how long', async () => {
    const user = userEvent.setup()
    const { onSelect } = renderQueue()

    const locked = row('1003')
    expect(within(locked).getByText('Picking · Carl Picker · 12 min')).toBeInTheDocument()
    expect(within(locked).queryByRole('button', { name: /#1003/ })).not.toBeInTheDocument()
    expect(within(locked).queryByRole('button', { name: /Take over/ })).not.toBeInTheDocument()
    await user.click(within(locked).getByText('#1003'))

    expect(onSelect).not.toHaveBeenCalled()
  })

  it('an admin may take over an order someone else holds', async () => {
    const user = userEvent.setup()
    const { onSelect } = renderQueue({ isAdmin: true })

    await user.click(within(row('1003')).getByRole('button', { name: 'Take over #1003' }))

    expect(onSelect).toHaveBeenCalledWith(entries[2].sales_order)
  })

  it('an order the current user holds stays workable', async () => {
    const user = userEvent.setup()
    const { onSelect } = renderQueue()

    await user.click(within(row('1004')).getByRole('button', { name: /#1004/ }))

    expect(onSelect).toHaveBeenCalledWith(entries[3].sales_order)
  })

  it('selecting a free order calls onSelect', async () => {
    const user = userEvent.setup()
    const { onSelect } = renderQueue()

    await user.click(screen.getByRole('button', { name: /#1001/ }))

    expect(onSelect).toHaveBeenCalledWith(entries[0].sales_order)
  })

  it('pick list button is disabled until orders are ticked, then opens with the ticked ids', async () => {
    const user = userEvent.setup()
    const { onOpenPickList, onSelect } = renderQueue()
    const pickListButton = screen.getByRole('button', { name: /Pick list/ })
    expect(pickListButton).toBeDisabled()

    await user.click(screen.getByRole('checkbox', { name: 'Add #1001 to pick list' }))
    await user.click(screen.getByRole('checkbox', { name: 'Add #1003 to pick list' }))
    expect(pickListButton).toHaveTextContent('Pick list (2)')
    await user.click(pickListButton)

    expect(onOpenPickList).toHaveBeenCalledWith(['so-1', 'so-3'])
    expect(onSelect).not.toHaveBeenCalled()
  })

  it('select all ticks every visible order', async () => {
    const user = userEvent.setup()
    const { onOpenPickList } = renderQueue()

    await user.click(screen.getByRole('checkbox', { name: 'Select all orders' }))
    await user.click(screen.getByRole('button', { name: /Pick list/ }))

    expect(onOpenPickList).toHaveBeenCalledWith(['so-1', 'so-2', 'so-3', 'so-4'])
  })

  it('search narrows the list', async () => {
    const user = userEvent.setup()
    renderQueue()

    await user.type(screen.getByPlaceholderText('Search orders...'), 'bo ng')

    expect(screen.queryByText('#1001')).not.toBeInTheDocument()
    expect(screen.getByText('#1002')).toBeInTheDocument()
  })

  it('shows loading, error and empty states', () => {
    const props = { loading: false, error: '', activeOrderId: null, activeInProgress: false, currentUserId: 'me', isAdmin: false, nameFor, onSelect: vi.fn(), onOpenPickList: vi.fn() }
    const { rerender } = render(<PackingOrderQueue {...props} entries={[]} loading />)
    expect(screen.getByText('Loading…')).toBeInTheDocument()

    rerender(<PackingOrderQueue {...props} entries={[]} error="boom" />)
    expect(screen.getByText('boom')).toBeInTheDocument()

    rerender(<PackingOrderQueue {...props} entries={[]} />)
    expect(screen.getByText('No orders to pack')).toBeInTheDocument()
  })
})

describe('PackingOrderQueue with amounts hidden', () => {
  it('never nests a button inside another (the hidden total is plain text)', () => {
    render(
      <AmountVisibilityProvider>
        <PackingOrderQueue
          entries={entries} loading={false} error="" activeOrderId={null} activeInProgress={false}
          currentUserId="me" isAdmin={false} nameFor={nameFor}
          onSelect={vi.fn()} onOpenPickList={vi.fn()}
        />
      </AmountVisibilityProvider>,
    )
    for (const button of screen.getAllByRole('button')) {
      expect(button.querySelector('button')).toBeNull()
    }
  })
})
