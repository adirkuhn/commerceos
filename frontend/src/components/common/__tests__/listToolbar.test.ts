import { describe, it, expect } from 'vitest'
import * as toolbar from '../listToolbar'

describe('list toolbar classes', () => {
  it('apply unconditionally -- there is one layout, no variant prefix', () => {
    for (const [name, classes] of Object.entries(toolbar)) {
      expect(classes, name).not.toMatch(/(^|\s)[a-z-]+:/)
    }
  })

  it('dissolve rows into the band and keep its overrides winning', () => {
    expect(toolbar.LIST_TOOLBAR_ROW).toBe('!contents')
    expect(toolbar.LIST_TOOLBAR_ACTION).toContain('!ml-auto')
    expect(toolbar.LIST_TOOLBAR_TITLE).toBe('!hidden')
  })
})
