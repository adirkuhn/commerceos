import { describe, it, expect } from 'vitest'
import { injectPage } from '../html.ts'

describe('injectPage', () => {
  it('fills head, markup and state placeholders', () => {
    const out = injectPage('<head><!--app-head--></head><div><!--app-html--></div><script><!--app-state--></script>', {
      head: '<title>x</title>', html: '<p>hi</p>', state: '{"a":1}',
    })
    expect(out).toBe('<head><title>x</title></head><div><p>hi</p></div><script>{"a":1}</script>')
  })

  it('does not treat $ sequences in content as replacement patterns', () => {
    const out = injectPage('<!--app-html-->', { head: '', html: "<p>$& $' $1</p>", state: '' })
    expect(out).toBe("<p>$& $' $1</p>")
  })
})

describe('injectPage lang', () => {
  it('sets <html lang> from the page, sanitised', () => {
    const tpl = '<html lang="en"><head><!--app-head--></head><body><!--app-html--><!--app-state--></body></html>'
    expect(injectPage(tpl, { lang: 'pt-BR', head: '', html: '', state: '' })).toContain('<html lang="pt-BR">')
    expect(injectPage(tpl, { lang: 'x"><script>', head: '', html: '', state: '' })).toContain('<html lang="xscript">')
  })
})
