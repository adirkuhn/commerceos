import { describe, it, expect } from 'vitest'
import contract from '../../../components/contract.gen.json'
import { STYLE_CONTROLS, breakpointFor, controlsFor, setStyleValue } from '../styleControls'

describe('style controls', () => {
  it('offer every style property in the contract exactly once', () => {
    const offered = Object.values(STYLE_CONTROLS).flat().map(c => c.property)
    expect([...offered].sort()).toEqual(contract.style_properties)
  })

  it('only offer the groups a component allows', () => {
    const groups = controlsFor(['typography']).map(g => g.group)
    expect(groups).toEqual(['typography'])
  })

  it('map devices to breakpoints', () => {
    expect(breakpointFor('desktop')).toBe('base')
    expect(breakpointFor('tablet')).toBe('tablet')
    expect(breakpointFor('mobile')).toBe('mobile')
  })

  it('sets and clears values immutably', () => {
    const before = { base: { padding_y: 'lg' } }
    const after = setStyleValue(before, 'mobile', 'padding_y', 'sm')
    expect(after).toEqual({ base: { padding_y: 'lg' }, mobile: { padding_y: 'sm' } })
    expect(before).toEqual({ base: { padding_y: 'lg' } })
    expect(setStyleValue(after, 'mobile', 'padding_y', '')).toEqual({ base: { padding_y: 'lg' } })
  })
})
