// Parsers behind the enum contract test: they pull named string enums out
// of the backend's OpenAPI contracts and string-literal unions out of the
// frontend source, so the two can be compared by name.

// A named schema that is a string enum, in either YAML list form:
//     SalesOrderStatus:
//       type: string
//       enum: [pending, processing]      (or one "- value" per line)
const SCHEMA_ENUM = /\n {4}(\w+):\n {6}type: string\n(?: {6}(?!enum)\w+:.*\n)*? {6}enum:[ \t]*(\[[^\]]*\]|\n(?: {8}- .*\n)+)/g

const unquote = (v: string) => v.trim().replace(/^['"]|['"]$/g, '')

// Working copies may be CRLF (core.autocrlf) -- normalize before matching.
const lf = (s: string) => s.replace(/\r\n/g, '\n')

export function parseSpecEnums(yaml: string): Map<string, string[]> {
  const out = new Map<string, string[]>()
  for (const m of lf(yaml).matchAll(SCHEMA_ENUM)) {
    const list = m[2].trim()
    const values = list.startsWith('[')
      ? list.slice(1, -1).split(',').map(unquote)
      : list.split('\n').map(l => unquote(l.replace(/^\s*- /, '')))
    out.set(m[1], values)
  }
  return out
}

// `export type Name = 'a' | 'b'` (single- or multi-line, leading | allowed).
const LITERAL_UNION = /export type (\w+) =\s*((?:\|?\s*'[^']*'\s*)+)(?=\n|$|;)/g

export function parseLiteralUnions(ts: string): Map<string, string[]> {
  const out = new Map<string, string[]>()
  for (const m of lf(ts).matchAll(LITERAL_UNION)) {
    out.set(m[1], [...m[2].matchAll(/'([^']*)'/g)].map(v => v[1]))
  }
  return out
}
