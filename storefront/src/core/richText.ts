// Rich text is stored as a tiny JSON structure, never HTML, so a theme can
// never carry markup or scripts. Mirrors RichText in the theme contract.

export interface RichTextInline {
  text: string
  bold?: boolean
  italic?: boolean
  href?: string
}

export type RichTextBlock =
  | { type: 'paragraph'; children: RichTextInline[] }
  | { type: 'bulleted_list' | 'numbered_list'; items: RichTextInline[][] }

export type RichText = RichTextBlock[]

const SAFE_HREF = /^(https:\/\/|http:\/\/|\/|mailto:|tel:)/

/** A link target, or undefined when it isn't one of the allowed schemes. */
export function safeHref(href: string | undefined): string | undefined {
  return href && SAFE_HREF.test(href) ? href : undefined
}
