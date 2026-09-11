import { describe, expect, it } from 'vitest'
import { scopeCssToCanvas } from '../scopeCanvasCss'

describe('scopeCssToCanvas', () => {
  it('rewrites :root to the scope selector itself, not a descendant of it', () => {
    const result = scopeCssToCanvas(':root {\n  --brand-500: #1f8a3d;\n}', '#canvas')

    expect(result).toContain('#canvas {')
    expect(result).not.toContain('#canvas :root')
  })

  it('prefixes a tag selector list with the scope as a descendant combinator', () => {
    const result = scopeCssToCanvas("h1, h2 {\n  font-family: serif !important;\n}", '#canvas')

    expect(result).toBe('#canvas h1, #canvas h2 {\n  font-family: serif !important;\n}')
  })

  it('prefixes a class selector, preserving a chained pseudo-class', () => {
    const result = scopeCssToCanvas('.bg-brand-600:hover {\n  background-color: #c95344 !important;\n}', '#canvas')

    expect(result).toBe('#canvas .bg-brand-600:hover {\n  background-color: #c95344 !important;\n}')
  })

  it('scopes every rule in a multi-rule stylesheet independently', () => {
    const css = combinedTestCSS()
    const result = scopeCssToCanvas(css, '#canvas')

    expect(result).toContain('#canvas {')
    expect(result).toContain('#canvas h1, #canvas h2 {')
    expect(result).toContain('#canvas .rounded, #canvas .rounded-lg {')
  })

  it('leaves an at-rule unscoped rather than mis-splitting it', () => {
    const result = scopeCssToCanvas('@media (min-width: 640px) { .foo { color: red; } }', '#canvas')

    expect(result).toContain('@media')
  })

  function combinedTestCSS(): string {
    return ':root {\n  --brand-500: #1f8a3d;\n}\n\nh1, h2 {\n  font-family: serif !important;\n}\n.rounded, .rounded-lg {\n  border-radius: 1rem !important;\n}'
  }
})
