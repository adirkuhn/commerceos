// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { quoteDocumentHtml, type QuoteDocument } from '../quoteDocument'
import type { CompanyProfile } from '../../settings/company/types'

const seller: CompanyProfile = {
  legal_name: 'Cantinho Brasileiro B.V.', trading_name: 'Cantinho', vat_number: 'NL001', chamber_of_commerce: '1234',
  email: 'shop@cantinho.nl', phone: '', iban: '', updated_at: '',
  address: { street: 'Kerkstraat', house_number: '1', postal_code: '1011 AA', city: 'Amsterdam', country: 'NL' },
}

const doc = (over: Partial<QuoteDocument> = {}): QuoteDocument => ({
  seller, reference: 'Q-0007', name: 'Restaurant Sabor', customerName: 'Ana Souza', date: '24-10-2026', currency: 'EUR',
  lines: [
    { sku: 'FEI-1', product: 'Feijão', quantity: 100, unitPriceMinor: 871 },
    { sku: 'FAR-1', product: 'Farofa <500g>', quantity: 5, unitPriceMinor: 1250 },
  ],
  ...over,
})

const text = (html: string) => new DOMParser().parseFromString(html, 'text/html')

describe('quoteDocumentHtml', () => {
  it('is a full A4 document that opens the print dialog once loaded', () => {
    const html = quoteDocumentHtml(doc(), 'en')
    expect(html.startsWith('<!doctype html>')).toBe(true)
    expect(html).toContain('size: A4')
    expect(html).toContain('counter(page)')
    expect(html).toContain('display: table-header-group')
    expect(html).toContain('break-inside: avoid')
    expect(html).toContain('window.print()')
  })

  it('shows the seller, the quote number, date and who it is for', () => {
    const d = text(quoteDocumentHtml(doc(), 'en'))
    expect(d.title).toBe('Quote Q-0007 — Restaurant Sabor')
    const seller = d.querySelector('[data-part="seller"]')!.textContent!
    expect(seller).toContain('Cantinho Brasileiro B.V.')
    expect(seller).toContain('Kerkstraat 1')
    expect(seller).toContain('1011 AA Amsterdam')
    expect(seller).toContain('VAT: NL001')
    expect(d.querySelector('[data-part="meta"]')!.textContent).toContain('Q-0007')
    expect(d.querySelector('[data-part="meta"]')!.textContent).toContain('24-10-2026')
    expect(d.querySelector('[data-part="for"]')!.textContent).toBe('For: Restaurant Sabor · Ana Souza')
  })

  it('lists sku, product, qty, unit price and line total, then the order total', () => {
    const d = text(quoteDocumentHtml(doc(), 'en'))
    const rows = [...d.querySelectorAll('tr')].map(r => [...r.querySelectorAll('th, td')].map(c => c.textContent))
    expect(rows).toEqual([
      ['SKU', 'Product', 'Qty', 'Unit price', 'Line total'],
      ['FEI-1', 'Feijão', '100', '€8.71', '€871.00'],
      ['FAR-1', 'Farofa <500g>', '5', '€12.50', '€62.50'],
      ['Order total', '€933.50'],
    ])
  })

  it('escapes what it prints', () => {
    const html = quoteDocumentHtml(doc({ name: '<script>alert(1)</script>' }), 'en')
    expect(html).not.toContain('<script>alert(1)</script>')
    expect(html).toContain('&lt;script&gt;')
  })

  it('without a company profile, customer or number it leaves those out', () => {
    const d = text(quoteDocumentHtml(doc({ seller: null, customerName: null, reference: null, name: 'Draft' }), 'en'))
    expect(d.querySelector('[data-part="seller"]')).toBeNull()
    expect(d.querySelector('[data-part="for"]')!.textContent).toBe('For: Draft')
    expect(d.title).toBe('Quote — Draft')
  })
})
