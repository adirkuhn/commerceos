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
  it('puts partly packed lines on top and moves finished ones to the end after their flash', () => {
    vi.useFakeTimers()
    try {
      const props = {
        token: 'tok', sessionId: 's1', readOnly: false, onOpenLine: vi.fn(),
        lineItems: [line('a', 'Arroz', 1), line('b', 'Feijao', 2), line('c', 'Farofa', 1)],
      }
      const names = () => screen.getAllByRole('button').map(b => b.querySelector('p')?.textContent)
      const { rerender } = render(<PackingLineList {...props} packedByLineItem={new Map()} />)
      expect(names()).toEqual(['Arroz', 'Feijao', 'Farofa'])

      rerender(<PackingLineList {...props} packedByLineItem={new Map([['b', 1]])} />)
      expect(names()).toEqual(['Feijao', 'Arroz', 'Farofa'])

      rerender(<PackingLineList {...props} packedByLineItem={new Map([['b', 1], ['a', 1]])} />)
      expect(names()).toEqual(['Feijao', 'Arroz', 'Farofa'])
      expect(screen.getAllByRole('button')[1].className).toContain('ring-green-500')

      act(() => { vi.advanceTimersByTime(PACK_LINE_SETTLE_MS) })
      expect(names()).toEqual(['Feijao', 'Farofa', 'Arroz'])
    } finally {
      vi.useRealTimers()
    }
  })
})
