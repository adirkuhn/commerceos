import { describe, it, expect } from 'vitest'
import * as toolbar from '../listToolbar'

// Classic is the rollback design and must stay pixel-identical: everything
// these classes do outside depot is make the wrapper invisible
// (display:contents), so a page lays out exactly as before it was wrapped.
describe('list toolbar classes', () => {
  it('only style the depot design, apart from the invisible wrapper', () => {
    for (const [name, classes] of Object.entries(toolbar)) {
      const classicOnly = classes.split(/\s+/).filter(c => !c.startsWith('depot:'))
      expect(classicOnly, name).toEqual(name === 'LIST_TOOLBAR' ? ['contents'] : [])
    }
  })
})
