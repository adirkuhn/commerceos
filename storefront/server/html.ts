export interface PageParts {
  head: string
  html: string
  state: string
}

/** Fills index.html's placeholders. Function replacers keep `$` literal. */
export function injectPage(template: string, parts: PageParts): string {
  return template
    .replace('<!--app-head-->', () => parts.head)
    .replace('<!--app-html-->', () => parts.html)
    .replace('<!--app-state-->', () => parts.state)
}
