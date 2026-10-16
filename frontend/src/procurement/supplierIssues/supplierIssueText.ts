import type { TFunction } from 'i18next'
import type { SupplierIssue, SupplierIssueKind, SupplierIssueRef } from '../types'

export const SUPPLIER_ISSUE_KINDS: SupplierIssueKind[] = ['missing', 'damaged', 'lost']

// issueKey identifies an issue in a Set/Map (source + id, as the API does).
export const issueKey = (ref: SupplierIssueRef): string => `${ref.source}:${ref.source_id}`

// issueItemName names the item the way the supplier knows it: their own
// line text first, then ours.
export function issueItemName(issue: SupplierIssue): string {
  return issue.supplier_description || issue.product_name || issue.sku_code || issue.supplier_sku || '—'
}

// purchaseOrderRef is how a PO is quoted to a supplier: POs have no number,
// so the start of the id.
export const purchaseOrderRef = (id: string): string => `#${id.slice(0, 8).toUpperCase()}`

// defaultSelection: what still needs claiming -- everything not resolved.
export function defaultSelection(issues: SupplierIssue[]): Set<string> {
  return new Set(issues.filter(i => i.status !== 'resolved').map(issueKey))
}

export interface SupplierMessageContext {
  supplierName: string | null
  poRef: string
  // Already formatted in the tenant's date format.
  orderDate: string
}

// buildSupplierMessage writes the plain-text message for the supplier
// (copied, or the starting text of the email): items grouped by kind, each
// in the supplier's own words.
export function buildSupplierMessage(issues: SupplierIssue[], ctx: SupplierMessageContext, t: TFunction<'procurement'>): { subject: string; body: string } {
  const lines: string[] = [
    ctx.supplierName ? t('supplierIssues.message.greeting', { name: ctx.supplierName }) : t('supplierIssues.message.greetingNoName'),
    '',
    t('supplierIssues.message.intro', { ref: ctx.poRef, date: ctx.orderDate }),
  ]
  for (const kind of SUPPLIER_ISSUE_KINDS) {
    const ofKind = issues.filter(i => i.kind === kind)
    if (ofKind.length === 0) continue
    lines.push('', t(`supplierIssues.kind.${kind}`))
    for (const issue of ofKind) lines.push(messageLine(issue, t))
  }
  lines.push('', t('supplierIssues.message.closing'), '', t('supplierIssues.message.signOff'))
  return {
    subject: t('supplierIssues.message.subject', { ref: ctx.poRef }),
    body: lines.join('\n'),
  }
}

function messageLine(issue: SupplierIssue, t: TFunction<'procurement'>): string {
  let line = t('supplierIssues.message.line', { qty: issue.quantity, item: issueItemName(issue) })
  if (issue.supplier_sku) line += ` ${t('supplierIssues.message.yourCode', { code: issue.supplier_sku })}`
  const detail = issue.kind === 'missing' && issue.expected_qty != null
    ? t('supplierIssues.orderedReceived', { expected: issue.expected_qty, received: issue.received_qty ?? 0 })
    : issue.reason
  return detail ? `${line} — ${detail}` : line
}

// supplierIssuesCsv is the list as a spreadsheet; formatDate renders an ISO
// timestamp in the tenant's date format.
export function supplierIssuesCsv(issues: SupplierIssue[], t: TFunction<'procurement'>, formatDate: (iso: string) => string): string {
  const header = (['kind', 'item', 'supplierCode', 'product', 'sku', 'qty', 'ordered', 'received', 'reason', 'date', 'status', 'note'] as const)
    .map(col => t(`supplierIssues.col.${col}`))
  const rows = issues.map(i => [
    t(`supplierIssues.kind.${i.kind}`), i.supplier_description, i.supplier_sku, i.product_name ?? '', i.sku_code ?? '',
    String(i.quantity), i.expected_qty != null ? String(i.expected_qty) : '', i.received_qty != null ? String(i.received_qty) : '',
    i.reason, formatDate(i.occurred_at), t(`supplierIssues.status.${i.status}`), i.status_note,
  ])
  return [header, ...rows].map(row => row.map(cell => `"${cell.replace(/"/g, '""')}"`).join(',')).join('\n')
}
