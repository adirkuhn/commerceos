// Every ThemePage's custom_css (palette + starter-template overrides, or
// a merchant's own free-form CSS from CustomCSSPanel) is written to run
// unscoped on storefront-frontend's own standalone document -- correct
// there, since :root/h1/.bg-brand-600 etc are meant to restyle that whole
// real page. ThemeEditorPage renders the same content inside its own
// admin canvas though, which shares a document with the editor's own
// chrome (a real <h1>, a real .bg-brand-600 Save button) -- injecting the
// same CSS unscoped there would restyle the editor itself, not just the
// canvas being previewed. This rewrites each rule's selector list to be a
// descendant of the given scope id instead, so the same source text stays
// confined to the canvas wherever it's previewed.
//
// Deliberately simple: a single non-nested selector/declaration regex, no
// @media/@supports handling. Every generated STARTER_TEMPLATE_CSS entry
// and paletteCustomCSS output is flat like this; an at-rule in a
// merchant's own free-form custom CSS is passed through unscoped rather
// than mis-split -- a rarer, lower-stakes case than the common one this
// exists for.
export function scopeCssToCanvas(css: string, scopeSelector: string): string {
  return css.replace(/([^{}]+)\{([^{}]*)\}/g, (match, selectors: string, declarations: string) => {
    if (selectors.trim().startsWith('@')) {
      return match
    }
    const scoped = selectors
      .split(',')
      .map(s => s.trim())
      .filter(Boolean)
      .map(s => (s === ':root' ? scopeSelector : `${scopeSelector} ${s}`))
      .join(', ')
    return `${scoped} {${declarations}}`
  })
}
