import { describe, it, expect } from 'vitest'
import { htmlToParagraphs } from '../plainText'

describe('htmlToParagraphs', () => {
  it('turns catalog HTML into plain paragraphs', () => {
    expect(htmlToParagraphs('<p>Soft &amp; warm</p><p>Line<br>two</p><script>alert(1)</script>')).toEqual(['Soft & warm', 'Line\ntwo'])
  })

  it('keeps plain text as is', () => {
    expect(htmlToParagraphs('One\n\nTwo')).toEqual(['One', 'Two'])
  })
})
