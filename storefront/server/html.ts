export interface PageParts {
  /** <html lang>; the template's own when absent. */
  lang?: string
  head: string
  html: string
  state: string
}

/** Fills index.html's placeholders. Function replacers keep `$` literal. */
export function injectPage(template: string, parts: PageParts): string {
  const withLang = parts.lang ? template.replace(/<html lang="[^"]*"/, () => `<html lang="${parts.lang!.replace(/[^A-Za-z0-9-]/g, '')}"`) : template
  return withLang
    .replace('<!--app-head-->', () => parts.head)
    .replace('<!--app-html-->', () => parts.html)
    .replace('<!--app-state-->', () => parts.state)
}
