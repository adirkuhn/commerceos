import type { RichText, RichTextBlock } from '../../core/richText'

// A simple text box for the Text component: blank lines separate
// paragraphs, "- " lines make a bulleted list, "1. " lines a numbered one.
// (A full rich-text editor with bold/italic/links is a later step.)

const BULLET = /^\s*[-*]\s+/
const NUMBER = /^\s*\d+[.)]\s+/

export function plainToRichText(plain: string): RichText {
  return plain.split(/\n\s*\n/).map(chunk => chunk.replace(/^\n+|\n+$/g, '')).filter(c => c.trim()).map((chunk): RichTextBlock => {
    const lines = chunk.split('\n')
    if (lines.every(l => BULLET.test(l))) return { type: 'bulleted_list', items: lines.map(l => [{ text: l.replace(BULLET, '') }]) }
    if (lines.every(l => NUMBER.test(l))) return { type: 'numbered_list', items: lines.map(l => [{ text: l.replace(NUMBER, '') }]) }
    return { type: 'paragraph', children: [{ text: chunk }] }
  })
}

export function richTextToPlain(content: RichText): string {
  const spans = (s: { text: string }[]) => s.map(i => i.text).join('')
  return content.map(b => {
    if (b.type === 'paragraph') return spans(b.children ?? [])
    return (b.items ?? []).map((item, i) => `${b.type === 'numbered_list' ? `${i + 1}.` : '-'} ${spans(item)}`).join('\n')
  }).join('\n\n')
}
