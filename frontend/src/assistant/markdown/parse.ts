// A deliberately small markdown subset for the assistant's answers:
// paragraphs, line breaks, **bold**, *italic*, `code`, bullet and numbered
// lists, and [text](path) links. It produces a tree the renderer turns into
// React elements -- never HTML -- so model output (which may echo pasted
// emails) can't inject markup.

export type Inline =
  | { kind: 'text'; text: string }
  | { kind: 'code'; text: string }
  | { kind: 'strong'; children: Inline[] }
  | { kind: 'em'; children: Inline[] }
  | { kind: 'link'; href: string; children: Inline[] }

export type Block =
  | { kind: 'paragraph'; lines: Inline[][] }
  | { kind: 'list'; ordered: boolean; items: Inline[][] }

const BULLET = /^\s*[-*]\s+(.*)$/
const NUMBERED = /^\s*\d+[.)]\s+(.*)$/

export function parseBlocks(text: string): Block[] {
  const blocks: Block[] = []
  let paragraph: Inline[][] | null = null
  let list: { ordered: boolean; items: Inline[][] } | null = null

  const flush = () => {
    if (paragraph) blocks.push({ kind: 'paragraph', lines: paragraph })
    if (list) blocks.push({ kind: 'list', ...list })
    paragraph = null
    list = null
  }

  for (const raw of text.replace(/\r\n?/g, '\n').split('\n')) {
    if (raw.trim() === '') {
      flush()
      continue
    }
    const bullet = BULLET.exec(raw)
    const numbered = bullet ? null : NUMBERED.exec(raw)
    const item = bullet ?? numbered
    if (item) {
      const ordered = numbered !== null
      if (!list || list.ordered !== ordered) {
        flush()
        list = { ordered, items: [] }
      }
      list.items.push(parseInline(item[1]))
      continue
    }
    if (list) flush()
    paragraph = paragraph ?? []
    paragraph.push(parseInline(raw))
  }
  flush()
  return blocks
}

// Earliest match wins; code first so nothing inside it is parsed.
const INLINE = /`([^`\n]+)`|\*\*([^*\n]+?)\*\*|\*([^*\s][^*\n]*?)\*|\[([^\]\n]+)\]\(([^)\s]+)\)/

export function parseInline(text: string): Inline[] {
  const out: Inline[] = []
  let rest = text
  while (rest) {
    const m = INLINE.exec(rest)
    if (!m) {
      out.push({ kind: 'text', text: rest })
      break
    }
    if (m.index > 0) out.push({ kind: 'text', text: rest.slice(0, m.index) })
    if (m[1] !== undefined) out.push({ kind: 'code', text: m[1] })
    else if (m[2] !== undefined) out.push({ kind: 'strong', children: parseInline(m[2]) })
    else if (m[3] !== undefined) out.push({ kind: 'em', children: parseInline(m[3]) })
    else out.push({ kind: 'link', href: m[5], children: parseInline(m[4]) })
    rest = rest.slice(m.index + m[0].length)
  }
  return out
}
