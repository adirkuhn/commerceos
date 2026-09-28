import { describe, it, expect } from 'vitest'
import { checkBudget, initialScripts } from '../budget.ts'

describe('initialScripts', () => {
  it('finds the entry script and its preloads', () => {
    const html = `<script type="module" crossorigin src="/store/assets/runtime-a.js"></script>
      <link rel="modulepreload" crossorigin href="/store/assets/renderers-b.js"><link rel="stylesheet" href="/store/assets/x.css">`
    expect(initialScripts(html)).toEqual(['/store/assets/runtime-a.js', '/store/assets/renderers-b.js'])
  })
})

describe('checkBudget', () => {
  it('passes a small bundle', () => {
    const r = checkBudget([{ path: 'a.js', content: 'console.log(1)' }], 1024)
    expect(r.problems).toEqual([])
    expect(r.files[0].gzip).toBeGreaterThan(0)
  })

  it('fails over budget or when the editor leaks in', () => {
    const big = Array.from({ length: 5000 }, (_, i) => `v${i}=${Math.sin(i)}`).join(';')
    expect(checkBudget([{ path: 'a.js', content: big }], 1024).problems[0]).toMatch(/over the 1.0 KB budget/)
    expect(checkBudget([{ path: 'b.js', content: 'import "grapesjs"' }], 1024).problems).toEqual(['b.js contains the editor (GrapesJS)'])
  })
})
