// The CommerceOS blocks the email editor adds to the newsletter preset's
// own: email-safe (tables, inline styles only) and wired to the Message
// Center variables. Their colours are a default each tenant restyles in
// the editor. Lists repeat with data-each (see backend
// domain/textrender.ExpandEach), which survives the editor's HTML parser.
export type BlockLabelKey =
  | 'design.blocks.header' | 'design.blocks.greeting' | 'design.blocks.orderSummary'
  | 'design.blocks.orderItems' | 'design.blocks.orderTotals' | 'design.blocks.shippingAddress'
  | 'design.blocks.tracking' | 'design.blocks.stockItems' | 'design.blocks.footer'

export interface EmailBlock {
  id: string
  labelKey: BlockLabelKey
  content: string
}

// BlockTexts are the words inside the blocks, in the editing language
// (the merchant changes them freely afterwards).
export interface BlockTexts {
  hi: string
  thanks: string
  order: string
  product: string
  qty: string
  total: string
  subtotal: string
  shipping: string
  discount: string
  shippingTo: string
  track: string
  expires: string
  price: string
}

export const ENGLISH_BLOCK_TEXTS: BlockTexts = {
  hi: 'Hi', thanks: 'Thank you for your order.', order: 'Order', product: 'Product', qty: 'Qty', total: 'Total',
  subtotal: 'Subtotal', shipping: 'Shipping', discount: 'Discount', shippingTo: 'Shipping to', track: 'Track your package',
  expires: 'Expires', price: 'Price',
}

const BRAND = '#1f2937'
const ACCENT = '#2563eb'
const CELL = 'padding:8px;border-bottom:1px solid #e5e7eb;font-size:14px;color:#374151'
const FONT = 'font-family:Arial,Helvetica,sans-serif'

export function emailBlocks(x: BlockTexts): EmailBlock[] {
  return [
  {
    id: 'cos-header', labelKey: 'design.blocks.header',
    content: `<table width="100%" cellpadding="0" cellspacing="0" style="${FONT}"><tr><td style="background:${BRAND};padding:24px;color:#ffffff;font-size:22px;font-weight:bold">{{store_name}}</td></tr></table>`,
  },
  {
    id: 'cos-greeting', labelKey: 'design.blocks.greeting',
    content: `<div style="${FONT};padding:24px 24px 8px"><h2 style="margin:0 0 8px;font-size:20px;color:#111827">${x.hi} {{customer_name}},</h2><p style="margin:0;font-size:14px;color:#374151">${x.thanks}</p></div>`,
  },
  {
    id: 'cos-order-summary', labelKey: 'design.blocks.orderSummary',
    content: `<div style="${FONT};padding:8px 24px;font-size:14px;color:#374151"><strong>${x.order} #{{order_number}}</strong> · {{order_date}} · {{order_status_label}}</div>`,
  },
  {
    id: 'cos-order-items', labelKey: 'design.blocks.orderItems',
    content: `<table width="100%" cellpadding="0" cellspacing="0" style="${FONT};border-collapse:collapse"><tr><th align="left" style="${CELL}">${x.product}</th><th align="right" style="${CELL}">${x.qty}</th><th align="right" style="${CELL}">${x.total}</th></tr><tr data-each="items"><td style="${CELL}">{{name}}<br><span style="font-size:12px;color:#6b7280">{{sku}}</span></td><td align="right" style="${CELL}">{{quantity}}</td><td align="right" style="${CELL}">{{total}}</td></tr></table>`,
  },
  {
    id: 'cos-order-totals', labelKey: 'design.blocks.orderTotals',
    content: `<table width="100%" cellpadding="0" cellspacing="0" style="${FONT};font-size:14px;color:#374151"><tr><td style="padding:4px 8px">${x.subtotal}</td><td align="right" style="padding:4px 8px">{{subtotal}}</td></tr><tr><td style="padding:4px 8px">${x.shipping}</td><td align="right" style="padding:4px 8px">{{shipping_total}}</td></tr><tr><td style="padding:4px 8px">${x.discount}</td><td align="right" style="padding:4px 8px">{{discount_total}}</td></tr><tr><td style="padding:8px;font-weight:bold;border-top:2px solid #111827">${x.total}</td><td align="right" style="padding:8px;font-weight:bold;border-top:2px solid #111827">{{order_total}}</td></tr></table>`,
  },
  {
    id: 'cos-shipping-address', labelKey: 'design.blocks.shippingAddress',
    content: `<div style="${FONT};padding:16px 24px;font-size:14px;color:#374151"><strong>${x.shippingTo}</strong><br>{{shipping_name}}<br>{{shipping_address}}<br><span style="color:#6b7280">{{shipping_method}}</span></div>`,
  },
  {
    id: 'cos-tracking', labelKey: 'design.blocks.tracking',
    content: `<table cellpadding="0" cellspacing="0" style="${FONT};margin:16px 24px"><tr><td style="background:${ACCENT};border-radius:6px"><a href="{{tracking_url}}" style="display:inline-block;padding:12px 20px;color:#ffffff;font-size:14px;font-weight:bold;text-decoration:none">${x.track}</a></td></tr><tr><td style="padding-top:8px;font-size:12px;color:#6b7280">{{carrier}} · {{tracking_barcode}}</td></tr></table>`,
  },
  {
    id: 'cos-stock-items', labelKey: 'design.blocks.stockItems',
    content: `<table width="100%" cellpadding="0" cellspacing="0" style="${FONT};border-collapse:collapse"><tr><th style="${CELL}"></th><th align="left" style="${CELL}">${x.product}</th><th align="right" style="${CELL}">${x.qty}</th><th align="left" style="${CELL}">${x.expires}</th><th align="right" style="${CELL}">${x.price}</th></tr><tr data-each="items"><td style="${CELL};width:64px"><img src="{{image_url}}" alt="" width="56" style="display:block;width:56px;border-radius:6px"></td><td style="${CELL}">{{product_name}}<br><span style="font-size:12px;color:#6b7280">{{sku}}</span></td><td align="right" style="${CELL}">{{quantity}}</td><td style="${CELL}">{{expiry_date}} ({{days_left}})</td><td align="right" style="${CELL}">{{price}}</td></tr></table>`,
  },
  {
    id: 'cos-footer', labelKey: 'design.blocks.footer',
    content: `<div style="${FONT};padding:24px;font-size:12px;color:#6b7280;text-align:center;border-top:1px solid #e5e7eb">{{store_name}} · {{store_address}}<br>{{store_email}} · {{store_phone}}</div>`,
  },
  ]
}
