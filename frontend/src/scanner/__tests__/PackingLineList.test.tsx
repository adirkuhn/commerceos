// @vitest-environment jsdom
import { act, render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { PackingLineList } from '../PackingLineList'
import { PACK_LINE_SETTLE_MS } from '../usePackLineOrder'
import type { PickingSessionLineItem } from '../types'

vi.mock('../../pms/api', () => ({ listProductMedia: vi.fn().mockResolvedValue([]) }))

function line(id: string, name: string, quantity: number): PickingSessionLineItem {
  return { id, sku_id: `sku-${id}`, name, sku_code: id.toUpperCase(), barcode: '', quantity, price_minor_units: 100, total_minor_units: 100 * quantity }
}

describe('PackingLineList', () => {
  it('keeps the line just finished on top, green, until the next scan (same as the admin)', () => {
    vi.useFakeTimers()
    try {
      const props = {
        token: 'tok', sessionId: 's1', readOnly: false, onOpenLine: vi.fn(),
        lineItems: [line('a', 'Arroz', 1), line('b', 'Feijao', 2), line('c', 'Farofa', 1)],
      }
      const buttons = () => screen.getAllByRole('button')
      const names = () => buttons().map(b => b.querySelector('p')?.textContent)
      const { rerender } = render(<PackingLineList {...props} packedByLineItem={new Map()} />)
      expect(names()).toEqual(['Arroz', 'Feijao', 'Farofa'])

      rerender(<PackingLineList {...props} packedByLineItem={new Map([['b', 1]])} />)
      expect(names()).toEqual(['Feijao', 'Arroz', 'Farofa'])

      rerender(<PackingLineList {...props} packedByLineItem={new Map([['b', 1], ['a', 1]])} />)
      act(() => { vi.advanceTimersByTime(PACK_LINE_SETTLE_MS) })
      expect(names()).toEqual(['Arroz', 'Feijao', 'Farofa'])
      expect(buttons()[0].className).toContain('ring-green-500')

      rerender(<PackingLineList {...props} packedByLineItem={new Map([['b', 1], ['a', 1], ['c', 1]])} />)
      expect(names()).toEqual(['Farofa', 'Feijao', 'Arroz'])
      expect(buttons()[2].className).not.toContain('ring-green-500')
    } finally {
      vi.useRealTimers()
    }
  })
})
