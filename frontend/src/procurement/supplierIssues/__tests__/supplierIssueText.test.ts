import { describe, it, expect } from 'vitest'
import i18next from 'i18next'
import type { TFunction } from 'i18next'
import type { SupplierIssue } from '../../types'
import {
  buildSupplierMessage, defaultSelection, issueKey, issueItemName, purchaseOrderRef, supplierIssuesCsv,
} from '../supplierIssueText'

const t = i18next.getFixedT('en', 'procurement') as TFunction<'procurement'>

const missing: SupplierIssue = {
  source: 'booking_shortfall', source_id: 'line-1', kind: 'missing', sku_code: 'PDQ-1KG', product_name: 'Pão de queijo',
  supplier_sku: 'B-100', supplier_description: 'Pao de queijo 1kg', expected_qty: 10, received_qty: 7, quantity: 3,
  reason: '', reported_by: '', occurred_at: '2026-10-01T12:00:00Z', status: 'open', status_note: '',
}
const damaged: SupplierIssue = {
  source: 'booking_scan', source_id: 'scan-3', kind: 'damaged', supplier_sku: '', supplier_description: '',
  sku_code: 'GUA-2L', product_name: 'Guaraná 2L', quantity: 2, reason: 'crushed "box"', reported_by: 'ana',
  occurred_at: '2026-10-01T09:00:00Z', status: 'claimed', status_note: 'emailed',
}
const lost: SupplierIssue = {
  source: 'booking_scan', source_id: 'scan-4', kind: 'lost', supplier_sku: '', supplier_description: '',
  quantity: 1, reason: '', reported_by: '', occurred_at: '2026-10-01T09:00:00Z', status: 'resolved', status_note: 'CN-7',
}

describe('issueItemName', () => {
  it('prefers the supplier’s own text, then our product name, then a SKU', () => {
    expect(issueItemName(missing)).toBe('Pao de queijo 1kg')
    expect(issueItemName(damaged)).toBe('Guaraná 2L')
    expect(issueItemName({ ...lost, sku_code: 'X-1' })).toBe('X-1')
    expect(issueItemName(lost)).toBe('—')
  })
})

describe('purchaseOrderRef', () => {
  it('is a short, readable reference', () => {
    expect(purchaseOrderRef('3f6d1c2a-0b7e-4c1d-9a2f-5e8b7c6d4a31')).toBe('#3F6D1C2A')
  })
})

describe('defaultSelection', () => {
  it('selects everything not resolved yet', () => {
    expect(defaultSelection([missing, damaged, lost])).toEqual(new Set([issueKey(missing), issueKey(damaged)]))
  })
})

describe('buildSupplierMessage', () => {
  it('groups the items by kind in the supplier’s own words', () => {
    const { subject, body } = buildSupplierMessage([missing, damaged], { supplierName: 'Bauducco NL', poRef: '#3F6D1C2A', orderDate: '01/10/2026' }, t)

    expect(subject).toBe('Problems with purchase order #3F6D1C2A')
    expect(body).toBe([
      'Hello Bauducco NL,',
      '',
      'We found these problems with purchase order #3F6D1C2A of 01/10/2026:',
      '',
      'Missing',
      '- 3 × Pao de queijo 1kg (your code B-100) — ordered 10, received 7',
      '',
      'Damaged',
      '- 2 × Guaraná 2L — crushed "box"',
      '',
      'Please send us a credit note or the replacement items.',
      '',
      'Kind regards',
    ].join('\n'))
  })

  it('greets without a name when the PO has no supplier', () => {
    const { body } = buildSupplierMessage([lost], { supplierName: null, poRef: '#1', orderDate: '01/10/2026' }, t)

    expect(body.startsWith('Hello,\n')).toBe(true)
    expect(body).toContain('Lost\n- 1 × —')
  })
})

describe('supplierIssuesCsv', () => {
  it('has one row per issue, quoted', () => {
    const csv = supplierIssuesCsv([missing, damaged], t, iso => iso.slice(0, 10))
    const lines = csv.split('\n')

    expect(lines).toHaveLength(3)
    expect(lines[0]).toBe('"Type","Supplier item","Supplier code","Our product","Our SKU","Quantity","Ordered","Received","Reason","Date","Status","Note"')
    expect(lines[1]).toBe('"Missing","Pao de queijo 1kg","B-100","Pão de queijo","PDQ-1KG","3","10","7","","2026-10-01","Open",""')
    expect(lines[2]).toBe('"Damaged","","","Guaraná 2L","GUA-2L","2","","","crushed ""box""","2026-10-01","Claimed","emailed"')
  })
})
