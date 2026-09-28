import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { storePathFor } from '../editor/useDraft'

// Shoppers must never download the editor: nothing the storefront runtime
// (or the server) can import may reach GrapesJS or src/editor.
const root = join(import.meta.dirname, '..', '..')
const files = (dir: string): string[] =>
  readdirSync(dir).flatMap(f => {
    const p = join(dir, f)
    if (statSync(p).isDirectory()) return f === '__tests__' ? [] : files(p)
    return /\.(ts|tsx)$/.test(f) ? [p] : []
  })

describe('runtime bundle boundary', () => {
  const shopperCode = ['src/runtime', 'src/components', 'src/core', 'src/api', 'server'].flatMap(d => files(join(root, d)))

  it('finds the code it guards', () => {
    expect(shopperCode.length).toBeGreaterThan(10)
  })

  it.each(shopperCode.map(f => [relative(root, f), f]))('%s', (_name, file) => {
    const src = readFileSync(file, 'utf8')
    expect(src).not.toMatch(/from ['"]grapesjs/)
    expect(src).not.toMatch(/from ['"](\.\.?\/)+editor\//)
  })
})

describe('editor preview paths', () => {
  it.each([
    [{ kind: 'page', key: 'about' }, 'pages/about'],
    [{ kind: 'template', key: 'cart' }, 'cart'],
    [{ kind: 'template', key: 'home' }, ''],
    [{ kind: 'section_group', key: 'header' }, ''],
  ] as const)('%o -> %s', (doc, path) => {
    expect(storePathFor(doc)).toBe(path)
  })
})
