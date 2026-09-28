// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { RichTextView } from '../content/RichTextView'
import { rendererFor } from '../renderers'

describe('RichTextView', () => {
  it('renders marks, lists and safe links', () => {
    const { container } = render(<RichTextView content={[
      { type: 'paragraph', children: [{ text: 'Hi ', bold: true }, { text: 'there', italic: true, href: '/pages/about' }] },
      { type: 'numbered_list', items: [[{ text: 'one' }], [{ text: 'two' }]] },
    ]} />)
    expect(container.innerHTML).toBe('<p><strong>Hi </strong><a href="/pages/about"><em>there</em></a></p><ol><li>one</li><li>two</li></ol>')
  })

  it('drops script links and never interprets text as HTML', () => {
    const { container } = render(<RichTextView content={[
      { type: 'paragraph', children: [{ text: '<img src=x onerror=alert(1)>', href: 'javascript:alert(1)' }] },
    ]} />)
    expect(container.querySelector('a')).toBeNull()
    expect(container.querySelector('img')).toBeNull()
    expect(container.textContent).toBe('<img src=x onerror=alert(1)>')
  })

  it('opens external links safely', () => {
    const { container } = render(<RichTextView content={[{ type: 'paragraph', children: [{ text: 'x', href: 'https://example.com' }] }]} />)
    expect(container.querySelector('a')).toHaveAttribute('rel', 'noopener noreferrer')
  })
})

describe('renderers', () => {
  it('renders a heading at a clamped level with its node class', () => {
    const Heading = rendererFor('heading')!
    const { container } = render(<Heading node={{ id: 'h', type: 'heading' }} props={{ text: 'Hi', level: 9 }} className="n-h" />)
    expect(container.innerHTML).toBe('<h2 class="cos-heading n-h" data-node="h">Hi</h2>')
  })

  it('renders containers from their definition element', () => {
    const Section = rendererFor('section')!
    const { container } = render(<Section node={{ id: 's', type: 'section' }} props={{ full_width: false }} className="n-s"><span>x</span></Section>)
    expect(container.innerHTML).toBe('<section class="cos-section cos-contained n-s" data-node="s"><span>x</span></section>')
  })
})
