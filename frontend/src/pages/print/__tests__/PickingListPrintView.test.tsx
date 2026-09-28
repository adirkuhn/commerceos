// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PickingListPrintView } from '../PickingListPrintView'
import type { SalesOrder, SalesOrderLineItem } from '../../../oms/types'

const order = { id: 'so-1', number: '1001', external_order_id: 'ext-1', occurred_at: '2026-09-27T10:00:00Z' } as SalesOrder

function line(overrides: Partial<SalesOrderLineItem> = {}): SalesOrderLineItem {
  return { id: 'li-1', sku_id: 'sku-1', name: 'Tapioca 500g', sku_code: 'TAP-1', quantity: 6, price_minor_units: 458, ...overrides } as SalesOrderLineItem
}

beforeEach(() => {
  vi.spyOn(window, 'print').mockImplementation(() => {})
})

describe('PickingListPrintView', () => {
  it('prints the sold expiry date WooCommerce reported under the product', () => {
    render(<PickingListPrintView order={order} lineItems={[line({ sold_expiry_date: '2027-03-09' })]} onClose={vi.fn()} />)
    expect(screen.getByText('Sold exp. 09/03/2027')).toBeInTheDocument()
  })

  it('prints no sold expiry for a line without one', () => {
    render(<PickingListPrintView order={order} lineItems={[line()]} onClose={vi.fn()} />)
    expect(screen.getByText('Tapioca 500g')).toBeInTheDocument()
    expect(screen.queryByText(/Sold exp\./)).not.toBeInTheDocument()
  })
})
