import { describe, it, expect } from 'vitest'
import { parseLiteralUnions, parseSpecEnums } from '../enumContract'

const specFiles = import.meta.glob('../../../../backend/api/*/openapi.yaml', { query: '?raw', import: 'default', eager: true }) as Record<string, string>
const sourceFiles = import.meta.glob(['../../**/*.ts', '../../**/*.tsx', '!../../**/*.test.*', '!../../**/__tests__/**'], {
  query: '?raw', import: 'default', eager: true,
}) as Record<string, string>

describe('parsers', () => {
  it('reads inline and block-list schema enums', () => {
    const yaml = '\n    A:\n      type: string\n      enum: [x, y-z]\n    B:\n      type: string\n      description: b\n      enum:\n        - "p"\n        - q\n'
    expect(Object.fromEntries(parseSpecEnums(yaml))).toEqual({ A: ['x', 'y-z'], B: ['p', 'q'] })
  })
  it('reads single- and multi-line literal unions', () => {
    const ts = "export type A = 'x' | 'y'\nexport type B =\n  | 'p'\n  | 'q'\nexport type C = string\n"
    expect(Object.fromEntries(parseLiteralUnions(ts))).toEqual({ A: ['x', 'y'], B: ['p', 'q'] })
  })
  it('handles CRLF files', () => {
    expect(Object.fromEntries(parseSpecEnums('\r\n    A:\r\n      type: string\r\n      enum: [x]\r\n'))).toEqual({ A: ['x'] })
    expect(Object.fromEntries(parseLiteralUnions("export type A = 'x' | 'y'\r\n"))).toEqual({ A: ['x', 'y'] })
  })
})

// The frontend's API types are hand-written. Any string-literal union named
// like a schema in a backend OpenAPI contract must list exactly that
// schema's enum values -- a missing value renders unstyled/unlabelled or,
// in a <select>, silently shows as the first option; an extra one is a
// value the API never sends.
describe('frontend enums match the OpenAPI contracts', () => {
  const spec = new Map<string, { file: string; values: string[] }>()
  for (const [file, yaml] of Object.entries(specFiles)) {
    for (const [name, values] of parseSpecEnums(yaml)) spec.set(name, { file, values })
  }
  const frontend = new Map<string, { file: string; values: string[] }>()
  for (const [file, ts] of Object.entries(sourceFiles)) {
    for (const [name, values] of parseLiteralUnions(ts)) frontend.set(name, { file, values })
  }
  const shared = [...frontend.keys()].filter(n => spec.has(n)).sort()

  it('finds the contracts and the shared types (guards against a parser that silently matches nothing)', () => {
    expect(Object.keys(specFiles).length).toBeGreaterThan(20)
    expect(shared.length).toBeGreaterThanOrEqual(30)
  })

  it.each(shared)('%s', name => {
    const fe = frontend.get(name)!
    const api = spec.get(name)!
    expect([...fe.values].sort(), `${fe.file} vs ${api.file}`).toEqual([...api.values].sort())
  })
})
