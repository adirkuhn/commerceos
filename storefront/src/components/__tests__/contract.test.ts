import { describe, it, expect } from 'vitest'
import contract from '../contract.gen.json'
import { registry } from '../registry'

// contract.gen.json is exported from backend/api/theme/openapi.yaml (see
// backend/cmd/export-theme-catalog). The backend validates saved documents
// against that contract, so the registry must agree with it exactly.
describe('component registry matches the backend contract', () => {
  const byType = new Map(contract.components.map(c => [c.type, c]))

  it('registers exactly the contract components', () => {
    expect(registry.all().map(d => d.type).sort()).toEqual([...byType.keys()].sort())
  })

  it.each(registry.all().map(d => [d.type, d] as const))('%s', (_type, def) => {
    const c = byType.get(def.type)!
    expect(def.container).toBe(c.container)
    expect([...def.allowedChildren].sort()).toEqual(c.allowed_children)
    expect([...def.allowedParents].sort()).toEqual(c.allowed_parents)
    expect(def.dataContext).toBe(c.data_context)
    expect([...def.requiredIn].sort()).toEqual(c.required_in)
    expect(Object.keys(def.props).sort()).toEqual(c.props)
    for (const p of c.required_props) expect(def.defaults, `${def.type}.${p} default`).toHaveProperty(p)
  })
})
