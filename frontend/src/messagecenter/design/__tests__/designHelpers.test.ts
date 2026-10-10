import { describe, it, expect } from 'vitest'
import { emailBlocks, ENGLISH_BLOCK_TEXTS } from '../emailBlocks'

const EMAIL_BLOCKS = emailBlocks(ENGLISH_BLOCK_TEXTS)
import { previewTriggerFor, tokenFor, listSnippet, plainTextExcerpt } from '../designHelpers'
import type { TemplateVariable } from '../../types'

describe('EMAIL_BLOCKS', () => {
  it('has unique ids and label keys', () => {
    const ids = EMAIL_BLOCKS.map(b => b.id)
    expect(new Set(ids).size).toBe(ids.length)
    for (const b of EMAIL_BLOCKS) expect(b.labelKey).toMatch(/^design\.blocks\./)
  })

  it('repeats order lines with data-each so the editor keeps the table intact', () => {
    const items = EMAIL_BLOCKS.find(b => b.id === 'cos-order-items')!
    expect(items.content).toContain('data-each="items"')
    expect(items.content).toContain('{{name}}')
    expect(items.content).not.toContain('{{#items}}')
  })

  it('writes its words in the editing language', () => {
    const nl = emailBlocks({ ...ENGLISH_BLOCK_TEXTS, track: 'Volg je pakket' })
    expect(nl.find(b => b.id === 'cos-tracking')!.content).toContain('>Volg je pakket<')
  })

  it('uses only inline styles (email clients drop <style> blocks)', () => {
    for (const b of EMAIL_BLOCKS) expect(b.content).not.toContain('<style')
  })
})

describe('previewTriggerFor', () => {
  it('picks a sensible trigger per category', () => {
    expect(previewTriggerFor('order_status')).toBe('oms.sales_order.status_changed')
    expect(previewTriggerFor('tracking')).toBe('shipping.shipment.fulfillment_signal')
    expect(previewTriggerFor('stock_info')).toBe('inventory.stock.close_to_expiry_reported')
    expect(previewTriggerFor('purchase')).toBe('storefront.order.placed')
    expect(previewTriggerFor('custom')).toBe('storefront.order.placed')
  })
})

describe('tokenFor / listSnippet', () => {
  const items: TemplateVariable = { name: 'items', type: 'list', example: null, fields: [
    { name: 'name', type: 'text', example: 'A', fields: [] }, { name: 'quantity', type: 'text', example: 2, fields: [] },
  ] }

  it('writes a text variable as a token', () => {
    expect(tokenFor({ name: 'order_number', type: 'text', example: '1', fields: [] })).toBe('{{order_number}}')
  })

  it('writes a list as a repeating element listing its fields', () => {
    expect(listSnippet(items)).toBe('<div data-each="items" style="padding:4px 0">{{name}} · {{quantity}}</div>')
  })
})

describe('plainTextExcerpt', () => {
  it('turns HTML into a readable one-line excerpt', () => {
    expect(plainTextExcerpt('<div><h1>Hi&nbsp;{{customer_name}}</h1>\n<p>Your order &amp; more</p></div>')).toBe('Hi {{customer_name}} Your order & more')
    expect(plainTextExcerpt('plain text')).toBe('plain text')
  })
})
