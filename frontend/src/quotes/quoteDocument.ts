import i18next from 'i18next'
import { formatMoney } from '../format/money'
import type { CompanyProfile } from '../settings/company/types'
import type { ExportLine } from './quoteCsv'

// The quote for the customer as a standalone printable page (opened in its
// own tab, so the quote editor stays as it is): seller, number, date, who
// it's for, then sku, product, quantity, unit price and line totals, and the
// order total -- never cost or profit. "Save as PDF" in the print dialog
// makes the file. A4 with page numbers; the table header repeats on every
// page and a row never splits across two.

export interface QuoteDocument {
  seller: CompanyProfile | null
  reference: string | null
  name: string
  customerName: string | null
  // Already formatted in the shop's date format.
  date: string
  currency: string
  lines: ExportLine[]
}

const escapeHtml = (s: string) =>
  s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;')

function sellerLines(seller: CompanyProfile, t: (key: string, opts?: Record<string, unknown>) => string): string[] {
  const a = seller.address
  return [
    seller.legal_name || seller.trading_name,
    [a.street, a.house_number].filter(Boolean).join(' '),
    [a.postal_code, a.city].filter(Boolean).join(' '),
    seller.vat_number && t('orders:print.vat', { value: seller.vat_number }),
    seller.chamber_of_commerce && t('orders:print.coc', { value: seller.chamber_of_commerce }),
    seller.email,
    seller.phone,
  ].filter((l): l is string => Boolean(l))
}

const STYLE = `
  @page { size: A4; margin: 15mm 15mm 18mm; @bottom-right { content: counter(page) " / " counter(pages); font: 9pt system-ui, sans-serif; color: #6b7280; } }
  * { box-sizing: border-box; }
  body { margin: 0; font: 10pt/1.45 system-ui, -apple-system, "Segoe UI", Roboto, sans-serif; color: #111827; }
  .sheet { max-width: 180mm; margin: 0 auto; padding: 8mm 0; }
  header { display: flex; justify-content: space-between; gap: 12mm; padding-bottom: 6mm; border-bottom: 2px solid #111827; }
  .seller { font-size: 9pt; color: #374151; }
  .seller strong { display: block; font-size: 12pt; color: #111827; margin-bottom: 1mm; }
  .meta { text-align: right; }
  .meta h1 { margin: 0; font-size: 20pt; letter-spacing: 0.06em; }
  .meta p { margin: 1mm 0 0; color: #374151; }
  .for { margin: 6mm 0 5mm; font-size: 10.5pt; }
  table { width: 100%; border-collapse: collapse; }
  thead { display: table-header-group; }
  tfoot { display: table-row-group; }
  th { text-align: left; font-size: 8.5pt; text-transform: uppercase; letter-spacing: 0.04em; color: #4b5563; padding: 2mm 2mm; border-bottom: 1.5px solid #111827; }
  td { padding: 1.6mm 2mm; border-bottom: 1px solid #e5e7eb; vertical-align: top; }
  tr { break-inside: avoid; page-break-inside: avoid; }
  .num { text-align: right; font-variant-numeric: tabular-nums; white-space: nowrap; }
  .sku { font-family: ui-monospace, Menlo, Consolas, monospace; font-size: 8.5pt; color: #4b5563; white-space: nowrap; }
  tfoot td { border-bottom: none; border-top: 2px solid #111827; font-weight: 600; font-size: 11pt; padding-top: 3mm; }
  @media screen { body { background: #f3f4f6; } .sheet { background: #fff; margin: 8mm auto; padding: 12mm; box-shadow: 0 1px 4px rgba(0,0,0,.15); } }
`

export function quoteDocumentHtml(doc: QuoteDocument, language: string = i18next.language): string {
  // Untyped lookup: the keys span the quotes and orders namespaces.
  const t = (key: string, opts?: Record<string, unknown>) => i18next.t(key as never, { lng: language, ...opts }) as string
  const money = (minor: number) => escapeHtml(formatMoney(minor, doc.currency))
  const total = doc.lines.reduce((sum, l) => sum + l.quantity * l.unitPriceMinor, 0)
  const title = [t('quotes:print.title'), doc.reference].filter(Boolean).join(' ') + (doc.name.trim() ? ` — ${doc.name.trim()}` : '')
  const forLine = [doc.name.trim(), doc.customerName].filter(Boolean).join(' · ')
  const seller = doc.seller ? sellerLines(doc.seller, t) : []

  const rows = doc.lines.map(l => `
        <tr>
          <td class="sku">${escapeHtml(l.sku)}</td>
          <td>${escapeHtml(l.product)}</td>
          <td class="num">${l.quantity}</td>
          <td class="num">${money(l.unitPriceMinor)}</td>
          <td class="num">${money(l.quantity * l.unitPriceMinor)}</td>
        </tr>`).join('')

  return `<!doctype html>
<html lang="${escapeHtml(language)}">
<head>
<meta charset="utf-8">
<title>${escapeHtml(title)}</title>
<style>${STYLE}</style>
</head>
<body>
  <div class="sheet">
    <header>
      ${seller.length ? `<div class="seller" data-part="seller"><strong>${escapeHtml(seller[0])}</strong>${seller.slice(1).map(escapeHtml).join('<br>')}</div>` : '<div></div>'}
      <div class="meta" data-part="meta">
        <h1>${escapeHtml(t('quotes:print.title').toUpperCase())}</h1>
        ${doc.reference ? `<p>${escapeHtml(doc.reference)}</p>` : ''}
        <p>${escapeHtml(doc.date)}</p>
      </div>
    </header>
    ${forLine ? `<p class="for" data-part="for">${escapeHtml(t('quotes:print.for', { name: forLine, interpolation: { escapeValue: false } }))}</p>` : ''}
    <table>
      <thead>
        <tr>
          <th>${escapeHtml(t('quotes:print.sku'))}</th>
          <th>${escapeHtml(t('quotes:print.product'))}</th>
          <th class="num">${escapeHtml(t('quotes:print.quantity'))}</th>
          <th class="num">${escapeHtml(t('quotes:print.unitPrice'))}</th>
          <th class="num">${escapeHtml(t('quotes:print.lineTotal'))}</th>
        </tr>
      </thead>
      <tbody>${rows}
      </tbody>
      <tfoot>
        <tr><td colspan="4" class="num">${escapeHtml(t('quotes:print.orderTotal'))}</td><td class="num">${money(total)}</td></tr>
      </tfoot>
    </table>
  </div>
  <script>window.addEventListener('load', function () { window.focus(); window.print() })</script>
</body>
</html>`
}
