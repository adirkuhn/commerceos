import { describe, it, expect } from 'vitest'
import { readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { registry } from '../../../components/registry'
import { sortKeys, sourceTexts } from '../sourceTexts'

const file = join(import.meta.dirname, '..', '..', '..', 'i18n', 'locales', 'en', 'editor.json')
const SECTIONS = ['categories', 'components', 'style'] as const

// Component and style texts are written in code; en/editor.json carries a
// copy for translators. They must never drift. `npm run i18n:registry`
// (UPDATE_I18N_REGISTRY=1) rewrites the copy from the code.
describe('editor source texts', () => {
  it('en/editor.json matches the component registry and style controls', () => {
    const expected = sourceTexts(registry)
    const current = JSON.parse(readFileSync(file, 'utf8')) as Record<string, unknown>
    if (process.env.UPDATE_I18N_REGISTRY) {
      const next = sortKeys({ ...current, ...expected })
      writeFileSync(file, JSON.stringify(next, null, 2) + '\n')
      return
    }
    for (const s of SECTIONS) expect(current[s], `${s} -- run npm run i18n:registry`).toEqual(expected[s])
  })
})
