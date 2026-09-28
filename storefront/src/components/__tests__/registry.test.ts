import { describe, it, expect } from 'vitest'
import { ComponentRegistry, registry } from '../registry'
import { registeredRendererTypes } from '../renderers'
import type { ComponentDefinition } from '../types'

describe('ComponentRegistry', () => {
  it('rejects duplicate types', () => {
    const d = registry.get('heading')!
    expect(() => new ComponentRegistry([d, d])).toThrow(/duplicate/)
  })

  it('enforces placement rules', () => {
    expect(registry.canPlace('section', 'root')).toBe(true)
    expect(registry.canPlace('section', 'section')).toBe(false)
    expect(registry.canPlace('heading', 'section')).toBe(true)
    expect(registry.canPlace('heading', 'heading')).toBe(false)
    expect(registry.canPlace('marquee', 'section')).toBe(false)
  })

  it('has a renderer for every definition', () => {
    expect(registeredRendererTypes().sort()).toEqual(registry.all().map(d => d.type).sort())
  })

  it('gives every prop an editor field and a default', () => {
    for (const d of registry.all() as ComponentDefinition[]) {
      expect(Object.keys(d.props).sort(), d.type).toEqual(Object.keys(d.defaults).sort())
    }
  })

  it('gives every container an element', () => {
    for (const d of registry.all()) {
      expect(Boolean(d.element), d.type).toBe(d.container)
    }
  })
})
