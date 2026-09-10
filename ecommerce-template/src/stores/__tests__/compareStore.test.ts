import { describe, expect, it, beforeEach } from 'vitest'
import { useCompareStore, MAX_COMPARE_ITEMS } from '../compareStore'

beforeEach(() => useCompareStore.setState({ productIds: [] }))

describe('useCompareStore', () => {
  it('adds up to the max item limit and then stops', () => {
    for (let i = 0; i < MAX_COMPARE_ITEMS + 2; i++) useCompareStore.getState().add(`p${i}`)
    expect(useCompareStore.getState().productIds).toHaveLength(MAX_COMPARE_ITEMS)
  })

  it('clears all items', () => {
    useCompareStore.getState().add('p1')
    useCompareStore.getState().add('p2')
    useCompareStore.getState().clear()
    expect(useCompareStore.getState().productIds).toHaveLength(0)
  })
})
