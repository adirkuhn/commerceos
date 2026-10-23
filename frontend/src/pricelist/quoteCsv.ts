import type { PriceListRow } from './types'

export interface CsvLine {
  product: string
  quantity: number
  unitPriceMinor: number
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

/** The quote as CSV: product, quantity and unit price only. */
export function quoteCsv(lines: CsvLine[], headers: readonly [string, string, string], locale: string): string {
  const { field, decimal } = separators(locale)
  const rows = [
    [...headers],
    ...lines.map(l => [l.product, String(l.quantity), (l.unitPriceMinor / 100).toFixed(2).replace('.', decimal)]),
  ]
  return rows.map(r => r.map(v => cell(v, field)).join(field)).join('\r\n') + '\r\n'
}

export function productLabel(row: Pick<PriceListRow, 'product_name' | 'variant_name' | 'sku_code'>): string {
  const name = row.variant_name ? `${row.product_name} — ${row.variant_name}` : row.product_name
  return `${name} (${row.sku_code})`
}
