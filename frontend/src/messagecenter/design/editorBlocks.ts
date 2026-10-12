import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import type { LucideIcon } from 'lucide-react'
import { Calculator, CalendarClock, Hand, ListOrdered, MapPin, PanelBottom, PanelTop, ReceiptText, Truck } from 'lucide-react'
import type { BlockLabelKey, EmailBlock } from './emailBlocks'

// EditorBlock is a CommerceOS block as the editor shows it: a translated
// name and an icon tile (GrapesJS takes both as plain strings).
export interface EditorBlock {
  id: string
  label: string
  media: string
  content: string
}

const ICONS: Record<string, LucideIcon> = {
  'cos-header': PanelTop,
  'cos-greeting': Hand,
  'cos-order-summary': ReceiptText,
  'cos-order-items': ListOrdered,
  'cos-order-totals': Calculator,
  'cos-shipping-address': MapPin,
  'cos-tracking': Truck,
  'cos-stock-items': CalendarClock,
  'cos-footer': PanelBottom,
}

function iconMarkup(icon: LucideIcon): string {
  return renderToStaticMarkup(createElement(icon, { size: 24, strokeWidth: 1.75, 'aria-hidden': true }))
}

export function toEditorBlocks(blocks: EmailBlock[], translate: (key: BlockLabelKey) => string): EditorBlock[] {
  return blocks.map(b => ({
    id: b.id,
    label: translate(b.labelKey),
    media: iconMarkup(ICONS[b.id] ?? PanelTop),
    content: b.content,
  }))
}
