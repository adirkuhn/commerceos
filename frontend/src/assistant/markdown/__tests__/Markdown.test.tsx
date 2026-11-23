// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { Markdown } from '../Markdown'

const allowed = (path: string) => path.startsWith('/sales-orders/') || path.startsWith('/quotes/')

function md(text: string) {
  return render(<MemoryRouter><Markdown text={text} isAllowedPath={allowed} /></MemoryRouter>)
}

describe('Markdown', () => {
  it('renders paragraphs and keeps single line breaks', () => {
    const { container } = md('First line\nsecond line\n\nNew paragraph')

    const ps = container.querySelectorAll('p')
    expect(ps).toHaveLength(2)
    expect(ps[0].querySelector('br')).not.toBeNull()
    expect(ps[0]).toHaveTextContent('First linesecond line')
    expect(ps[1]).toHaveTextContent('New paragraph')
  })

  it('renders bold, italic and inline code', () => {
    const { container } = md('**CO-12** has *shipped* via `PostNL`')

    expect(container.querySelector('strong')).toHaveTextContent('CO-12')
    expect(container.querySelector('em')).toHaveTextContent('shipped')
    expect(container.querySelector('code')).toHaveTextContent('PostNL')
  })

  it('does not parse markdown inside inline code', () => {
    const { container } = md('`**not bold**`')

    expect(container.querySelector('strong')).toBeNull()
    expect(container.querySelector('code')).toHaveTextContent('**not bold**')
  })

  it('renders bullet and numbered lists', () => {
    const { container } = md('Orders:\n- CO-12\n* CO-13\n\n1. first\n2) second')

    const ul = container.querySelector('ul')
    expect(ul?.querySelectorAll('li')).toHaveLength(2)
    const ol = container.querySelector('ol')
    expect(ol?.querySelectorAll('li')).toHaveLength(2)
    expect(ol).toHaveTextContent('firstsecond')
  })

  it('links admin paths inside the app', () => {
    md('Order [CO-12](/sales-orders/abc-1) shipped.')

    const link = screen.getByRole('link', { name: 'CO-12' })
    expect(link).toHaveAttribute('href', '/sales-orders/abc-1')
  })

  it('formats inside links and list items', () => {
    md('- **Quote** [Q-0012](/quotes/q-1)')

    expect(screen.getByRole('link', { name: 'Q-0012' })).toHaveAttribute('href', '/quotes/q-1')
    expect(screen.getByText('Quote').tagName).toBe('STRONG')
  })

  it.each([
    ['external https', '[click](https://evil.example/steal)'],
    ['javascript:', '[click](javascript:alert(1))'],
    ['data:', '[click](data:text/html,<script>alert(1)</script>)'],
    ['protocol-relative', '[click](//evil.example)'],
    ['unknown admin path', '[click](/not-a-page/1)'],
    ['relative without slash', '[click](sales-orders/1)'],
  ])('renders a %s link as plain text', (_, text) => {
    const { container } = md(text)

    expect(screen.queryByRole('link')).toBeNull()
    expect(container).toHaveTextContent('click')
    expect(container.querySelector('a')).toBeNull()
  })

  it('never renders HTML from the text', () => {
    const { container } = md('<img src=x onerror="alert(1)"> <b>bold</b> <script>alert(1)</script>')

    expect(container.querySelector('img')).toBeNull()
    expect(container.querySelector('b')).toBeNull()
    expect(container.querySelector('script')).toBeNull()
    expect(container).toHaveTextContent('<b>bold</b>')
  })

  it('leaves unmatched markers as text', () => {
    const { container } = md('2 * 3 = 6 and **unclosed')

    expect(container.querySelector('em')).toBeNull()
    expect(container.querySelector('strong')).toBeNull()
    expect(container).toHaveTextContent('2 * 3 = 6 and **unclosed')
  })
})
