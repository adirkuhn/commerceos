// Product descriptions can arrive as HTML (e.g. imported from WooCommerce).
// The storefront never renders catalog HTML: it's turned into plain
// paragraphs and shown as text.

const ENTITIES: Record<string, string> = { amp: '&', lt: '<', gt: '>', quot: '"', '#39': "'", apos: "'", nbsp: ' ' }

export function htmlToParagraphs(input: string): string[] {
  const text = input
    .replace(/<(script|style)[^>]*>[\s\S]*?<\/\1>/gi, '')
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<\/(p|div|li|h[1-6])>/gi, '\n\n')
    .replace(/<[^>]+>/g, '')
    .replace(/&(amp|lt|gt|quot|#39|apos|nbsp);/g, (_, e: string) => ENTITIES[e])
  return text.split(/\n\s*\n/).map(p => p.trim()).filter(Boolean)
}
