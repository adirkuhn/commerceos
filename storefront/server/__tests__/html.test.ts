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
