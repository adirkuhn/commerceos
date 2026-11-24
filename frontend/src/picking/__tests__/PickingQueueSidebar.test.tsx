// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { PickingQueueSidebar } from '../PickingQueueSidebar'
import { AmountVisibilityProvider } from '../../format/amountVisibility'
import type { QueueEntry, SalesOrder } from '../../oms/types'

function order(overrides: Partial<SalesOrder>): SalesOrder {
  return {
    id: 'so-1', tenant_id: 't1', source: 'woocommerce', external_order_id: '501', number: '501', order_key: 'k',
    status: 'processing', payment_status: 'paid', currency_code: 'EUR', discount_total_minor_units: 0,
    shipping_total_minor_units: 0, tax_total_minor_units: 0, total_minor_units: 1000, occurred_at: '', created_at: '',
    ...overrides,
  }
}

const entries: QueueEntry[] = [
  { sales_order: order({ id: 'a', number: '501' }), lock: null, in_packing: false },
  { sales_order: order({ id: 'b', number: '502' }), lock: null, in_packing: false },
]

describe('PickingQueueSidebar with amounts hidden', () => {
  it('never nests a button inside another (the hidden total is plain text)', () => {
    render(
      <AmountVisibilityProvider>
        <PickingQueueSidebar
          entries={entries} loading={false} error="" activeOrderId={null} activeInProgress={false}
          viewId={null} nameFor={id => id} onSelect={vi.fn()}
        />
      </AmountVisibilityProvider>,
    )
    const rows = screen.getAllByRole('button', { pressed: false })
    expect(rows).toHaveLength(2)
    for (const row of rows) expect(row.querySelector('button')).toBeNull()
  })
})
