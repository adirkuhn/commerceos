import { describe, it, expect } from 'vitest'
import { plainToRichText, richTextToPlain } from '../richTextPlain'

describe('plain text <-> rich text', () => {
  it('splits paragraphs on blank lines and turns "- " lines into lists', () => {
    expect(plainToRichText('Hello there\nsecond line\n\n- one\n- two\n\n1. first\n2. second')).toEqual([
      { type: 'paragraph', children: [{ text: 'Hello there\nsecond line' }] },
      { type: 'bulleted_list', items: [[{ text: 'one' }], [{ text: 'two' }]] },
      { type: 'numbered_list', items: [[{ text: 'first' }], [{ text: 'second' }]] },
    ])
  })

  it('round-trips', () => {
    const plain = 'A\n\n- b\n- c'
    expect(richTextToPlain(plainToRichText(plain))).toBe(plain)
  })

  it('reads marks and links as their text', () => {
    expect(richTextToPlain([{ type: 'paragraph', children: [{ text: 'x', bold: true }, { text: 'y', href: '/a' }] }])).toBe('xy')
  })

  it('treats empty input as no content', () => {
    expect(plainToRichText('  \n\n ')).toEqual([])
  })
})
