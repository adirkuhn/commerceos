import type { MessageCategory, TemplateVariable, TriggerSubject } from '../types'

// previewTriggerFor is the trigger whose sample data a template of this
// category previews with, until the user picks another.
export function previewTriggerFor(category: MessageCategory): TriggerSubject {
  switch (category) {
    case 'order_status': return 'oms.sales_order.status_changed'
    case 'tracking': return 'shipping.shipment.fulfillment_signal'
    case 'stock_info': return 'inventory.stock.close_to_expiry_reported'
    default: return 'storefront.order.placed'
  }
}

export function tokenFor(v: TemplateVariable): string {
  return `{{${v.name}}}`
}

// listSnippet is a list variable as an element repeated once per entry
// (data-each, which the backend renderer expands).
export function listSnippet(v: TemplateVariable): string {
  const fields = v.fields.map(f => `{{${f.name}}}`).join(' · ')
  return `<div data-each="${v.name}" style="padding:4px 0">${fields}</div>`
}

const ENTITIES: Record<string, string> = { '&nbsp;': ' ', '&amp;': '&', '&lt;': '<', '&gt;': '>', '&quot;': '"', '&#39;': "'" }

// plainTextExcerpt is an HTML body as one readable line, for lists.
export function plainTextExcerpt(html: string): string {
  return html
    .replace(/<style[\s\S]*?<\/style>/gi, ' ')
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;|&amp;|&lt;|&gt;|&quot;|&#39;/g, m => ENTITIES[m])
    .replace(/\s+/g, ' ')
    .trim()
}
