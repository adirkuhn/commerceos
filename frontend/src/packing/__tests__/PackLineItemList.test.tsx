// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { PackLineItemList } from '../PackLineItemList'
import type { PickingSessionLineItem } from '../../scanner/types'

vi.mock('../../pms/api', () => ({ listProductMedia: vi.fn().mockResolvedValue([]) }))

function line(overrides: Partial<PickingSessionLineItem> = {}): PickingSessionLineItem {
  return { id: 'li-1', sku_id: 'sku-1', name: 'Tapioca 500g', sku_code: 'TAP-1', barcode: '789', quantity: 6, price_minor_units: 458, total_minor_units: 2747, ...overrides }
}

function renderList(lineItems: PickingSessionLineItem[]) {
  render(<PackLineItemList token="tok" lineItems={lineItems} packedByLineItem={new Map()} readOnly={false} onOpenLine={vi.fn()} />)
}

describe('PackLineItemList', () => {
  it('shows the sold expiry date WooCommerce reported for a line', () => {
    renderList([line({ sold_expiry_date: '2027-03-09' })])
    expect(screen.getByText('Sold exp. 09/03/2027')).toBeInTheDocument()
  })

  it('shows no sold expiry for a line without one', () => {
    renderList([line()])
    expect(screen.queryByText(/Sold exp\./)).not.toBeInTheDocument()
  })
})
