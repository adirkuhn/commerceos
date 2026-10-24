import type { PriceListRow } from './types'

export interface ExportLine {
  sku: string
  product: string
  quantity: number
  unitPriceMinor: number
}

export interface ExportHeaders {
  sku: string
  product: string
  quantity: string
  unitPrice: string
  lineTotal: string
  total: string
}

// Excel reads CSV with the computer's list separator: where the decimal
// mark is a comma (nl, pt-BR) that is ";" -- a plain comma file would open
// as one column there.
function separators(locale: string): { field: string; decimal: string } {
  const decimal = new Intl.NumberFormat(locale).format(1.5).includes(',') ? ',' : '.'
  return { field: decimal === ',' ? ';' : ',', decimal }
}

function cell(value: string, field: string): string {
  return value.includes(field) || /["\r\n]/.test(value) ? `"${value.replace(/"/g, '""')}"` : value
}

/** The quote as CSV: sku, product, quantity, unit price, line total, then the order total. */
export function quoteCsv(lines: ExportLine[], headers: ExportHeaders, locale: string): string {
  const { field, decimal } = separators(locale)
  const amount = (minor: number) => (minor / 100).toFixed(2).replace('.', decimal)
  const total = lines.reduce((sum, l) => sum + l.quantity * l.unitPriceMinor, 0)
  const rows = [
    [headers.sku, headers.product, headers.quantity, headers.unitPrice, headers.lineTotal],
    ...lines.map(l => [l.sku, l.product, String(l.quantity), amount(l.unitPriceMinor), amount(l.quantity * l.unitPriceMinor)]),
    ['', '', '', headers.total, amount(total)],
  ]
  return rows.map(r => r.map(v => cell(v, field)).join(field)).join('\r\n') + '\r\n'
}

/** The product's name; its variant only when that says something the name doesn't. */
export function productName(row: Pick<PriceListRow, 'product_name' | 'variant_name'>): string {
  return row.variant_name && row.variant_name !== row.product_name ? `${row.product_name} — ${row.variant_name}` : row.product_name
}
