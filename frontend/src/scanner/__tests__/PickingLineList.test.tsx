// @vitest-environment jsdom
import { act, render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { PickingLineList } from '../PickingLineList'
import { PACK_LINE_SETTLE_MS } from '../usePackLineOrder'
import type { PickingSessionLineItem } from '../types'

vi.mock('../../pms/api', () => ({ listProductMedia: vi.fn().mockResolvedValue([]) }))

function line(id: string, name: string, quantity: number, linked = true): PickingSessionLineItem {
  return { id, sku_id: linked ? `sku-${id}` : null, name, sku_code: id.toUpperCase(), barcode: '', quantity, price_minor_units: 100, total_minor_units: 100 * quantity } as PickingSessionLineItem
}

describe('PickingLineList', () => {
  it('keeps the line just picked on top, green, until the next scan (same as packing)', () => {
    vi.useFakeTimers()
    try {
      const props = {
        token: 'tok', sessionId: 's1', readOnly: false, onOpenLine: vi.fn(),
        lineItems: [line('a', 'Arroz', 1), line('b', 'Feijao', 2), line('c', 'Farofa', 1)],
      }
      const buttons = () => screen.getAllByRole('button')
      const names = () => buttons().map(b => b.querySelector('p')?.textContent)
      const { rerender } = render(<PickingLineList {...props} pickedByLineItem={new Map()} />)
      expect(names()).toEqual(['Arroz', 'Feijao', 'Farofa'])

      rerender(<PickingLineList {...props} pickedByLineItem={new Map([['b', 1]])} />)
      expect(names()).toEqual(['Feijao', 'Arroz', 'Farofa'])

      rerender(<PickingLineList {...props} pickedByLineItem={new Map([['b', 1], ['a', 1]])} />)
      act(() => { vi.advanceTimersByTime(PACK_LINE_SETTLE_MS) })
      expect(names()).toEqual(['Arroz', 'Feijao', 'Farofa'])
      expect(buttons()[0].className).toContain('ring-green-500')

      rerender(<PickingLineList {...props} pickedByLineItem={new Map([['b', 1], ['a', 1], ['c', 1]])} />)
      expect(names()).toEqual(['Farofa', 'Feijao', 'Arroz'])
      expect(buttons()[2].className).not.toContain('ring-green-500')
    } finally {
      vi.useRealTimers()
    }
  })

  it('opens the tapped line with what is picked so far', () => {
    const onOpenLine = vi.fn()
    const b = line('b', 'Feijao', 2)
    render(<PickingLineList token="tok" sessionId="s1" readOnly={false} onOpenLine={onOpenLine} lineItems={[b]} pickedByLineItem={new Map([['b', 1]])} />)

    act(() => { screen.getByRole('button').click() })

    expect(onOpenLine).toHaveBeenCalledWith(b, 1)
  })

  it('says when a line is not linked to a product', () => {
    render(<PickingLineList token="tok" sessionId="s1" readOnly={false} onOpenLine={vi.fn()} lineItems={[line('x', 'Mystery', 1, false)]} pickedByLineItem={new Map()} />)

    expect(screen.getByText('Not linked to a product yet')).toBeInTheDocument()
  })
})
