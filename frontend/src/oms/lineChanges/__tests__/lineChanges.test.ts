import { describe, it, expect } from 'vitest'
import { groupLineChanges, changeProductLabel, isExtraItem } from '../lineChanges'
import { change } from './fixtures'

describe('groupLineChanges', () => {
  it('groups changes under the line they touch, in order', () => {
    const a = change({ id: 'a' })
    const b = change({ id: 'b', action: 'refund' })
    const c = change({ id: 'c', line_item_id: 'line-2' })

    const { byLine, notAdded } = groupLineChanges([a, b, c])

    expect(byLine.get('line-1')).toEqual([a, b])
    expect(byLine.get('line-2')).toEqual([c])
    expect(notAdded).toEqual([])
  })

  it('keeps extras that never became a line apart', () => {
    const skipped = change({ id: 'x', action: 'freebie', line_item_id: null })

    expect(groupLineChanges([skipped]).notAdded).toEqual([skipped])
  })
})

describe('changeProductLabel', () => {
  it('shows the product, and the variant only when it says something else', () => {
    expect(changeProductLabel(change({ product_name: 'Feijão', variant_name: 'Feijão' }))).toBe('Feijão')
    expect(changeProductLabel(change({ product_name: 'Feijão', variant_name: '5kg' }))).toBe('Feijão — 5kg')
  })

  it('falls back to the typed name, then the SKU code, when the SKU is gone', () => {
    expect(changeProductLabel(change({ product_name: '', reason: 'Guaraná' }))).toBe('Guaraná')
    expect(changeProductLabel(change({ product_name: '', reason: '', sku_code: 'GUA-1' }))).toBe('GUA-1')
  })
})

describe('isExtraItem', () => {
  it('is true for free and charged extras only', () => {
    expect(isExtraItem({ action: 'freebie' })).toBe(true)
    expect(isExtraItem({ action: 'new_item' })).toBe(true)
    expect(isExtraItem({ action: 'substitute' })).toBe(false)
    expect(isExtraItem({ action: 'refund' })).toBe(false)
  })
})
