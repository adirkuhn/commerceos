// @vitest-environment jsdom
import { act, render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { PACK_LINE_SETTLE_MS } from '../../scanner/usePackLineOrder'
import { PackLineItemList } from '../PackLineItemList'
import type { PickingSessionLineItem } from '../../scanner/types'

vi.mock('../../pms/api', () => ({ listProductMedia: vi.fn().mockResolvedValue([]) }))

function line(overrides: Partial<PickingSessionLineItem> = {}): PickingSessionLineItem {
  return { id: 'li-1', sku_id: 'sku-1', name: 'Tapioca 500g', sku_code: 'TAP-1', barcode: '789', quantity: 6, price_minor_units: 458, total_minor_units: 2747, ...overrides }
}

function renderList(lineItems: PickingSessionLineItem[]) {
  render(<PackLineItemList token="tok" sessionId="s1" lineItems={lineItems} packedByLineItem={new Map()} readOnly={false} onOpenLine={vi.fn()} />)
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

  it('moves a partly packed line to the top and a finished one to the end, after its flash', () => {
    vi.useFakeTimers()
    try {
      const lines = [
        line({ id: 'a', name: 'Arroz', quantity: 1 }),
        line({ id: 'b', name: 'Feijao', quantity: 2 }),
        line({ id: 'c', name: 'Farofa', quantity: 1 }),
      ]
      const props = { token: 'tok', sessionId: 's1', lineItems: lines, readOnly: false, onOpenLine: vi.fn() }
      const names = () => screen.getAllByTestId('pack-line').map(el => el.dataset.lineId)
      const { rerender } = render(<PackLineItemList {...props} packedByLineItem={new Map()} />)
      expect(names()).toEqual(['a', 'b', 'c'])

      rerender(<PackLineItemList {...props} packedByLineItem={new Map([['b', 1]])} />)
      expect(names()).toEqual(['b', 'a', 'c'])

      rerender(<PackLineItemList {...props} packedByLineItem={new Map([['b', 1], ['a', 1]])} />)
      expect(names()).toEqual(['b', 'a', 'c'])
      expect(screen.getAllByTestId('pack-line')[1].dataset.highlight).toBe('done')

      act(() => { vi.advanceTimersByTime(PACK_LINE_SETTLE_MS) })
      expect(names()).toEqual(['b', 'c', 'a'])
    } finally {
      vi.useRealTimers()
    }
  })
})
