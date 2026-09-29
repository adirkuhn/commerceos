import { describe, it, expect } from 'vitest'
import ts from 'typescript'

// Modules already translated. Their UI text must come from t(): this test
// flags text written straight into JSX and literal user-facing attributes.
// Add a module here once it's translated.
export const TRANSLATED = [
  'components/layout',
]

// Every component source file, as text, keyed by path relative to src/.
const SOURCES = Object.fromEntries(
  Object.entries(import.meta.glob<string>(['../../**/*.tsx', '!../../**/__tests__/**', '!../../**/*.test.tsx'], { query: '?raw', import: 'default', eager: true }))
    .map(([path, text]) => [path.replace(/^\.\.\/\.\.\//, ''), text]),
)

// Words that are the same in every language (brand, units, symbols).
const ALLOWED = /^(CommerceOS|SKU|SKUs|EAN|GTIN|API|URL|CSV|PDF|ID|OK|K|ESC|FEFO|FIFO|RFM|€|%|—|–|·|×|\+|-|#|\/|:|\(|\)|[0-9.,:%€$ ]+)$/

const ATTRIBUTES = new Set(['placeholder', 'title', 'aria-label', 'label', 'alt'])

/** Text written straight into JSX, and literal user-facing attributes. */
export function hardcodedTexts(source: string): string[] {
  const file = ts.createSourceFile('x.tsx', source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX)
  const found: string[] = []
  const visit = (node: ts.Node) => {
    if (ts.isJsxText(node)) {
      const text = node.text.replace(/\s+/g, ' ').trim()
      if (text && /[A-Za-zÀ-ÿ]/.test(text) && !ALLOWED.test(text)) found.push(text)
    } else if (ts.isJsxAttribute(node) && node.initializer && ts.isStringLiteral(node.initializer)) {
      const name = node.name.getText(file)
      const value = node.initializer.text
      if (ATTRIBUTES.has(name) && /[A-Za-z]/.test(value) && !ALLOWED.test(value)) found.push(`${name}="${value}"`)
    }
    ts.forEachChild(node, visit)
  }
  visit(file)
  return found
}

describe('translated modules have no hard-coded UI text', () => {
  it.each(TRANSLATED)('%s', mod => {
    const inModule = Object.entries(SOURCES).filter(([path]) => path.startsWith(`${mod}/`))
    expect(inModule.length, `no .tsx files under ${mod}`).toBeGreaterThan(0)
    const offenders = inModule.flatMap(([path, text]) => hardcodedTexts(text).map(t => `${path}: ${t}`))
    expect(offenders).toEqual([])
  })
})

describe('hardcodedTexts', () => {
  it('finds JSX text and literal attributes, ignoring expressions and symbols', () => {
    expect(hardcodedTexts('<p>Loading…</p><b>{t("x")}</b><i>€</i><span title="Hi there" aria-label={t("y")}>')).toEqual(['Loading…', 'title="Hi there"'])
  })
})
