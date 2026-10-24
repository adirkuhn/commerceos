import { describe, it, expect } from 'vitest'
import { lineMoney } from '../lineMoney'
import type { SalesOrderLineItem } from '../../types'

function line(over: Partial<SalesOrderLineItem>): SalesOrderLineItem {
  return {
    id: 'li-1', name: 'Caldo de Cana', sku_code: '', barcode: '', quantity: 3, price_minor_units: 458,
    subtotal_minor_units: 1373, subtotal_tax_minor_units: 124, total_minor_units: 1373, total_tax_minor_units: 124, ...over,
  }
}

describe('lineMoney', () => {
  it('reads the totals the backend computed', () => {
    expect(lineMoney(line({ total_excl_tax_minor_units: 1373, total_incl_tax_minor_units: 1497 })))
      .toEqual({ exclTax: 1373, tax: 124, inclTax: 1497, unitInclTax: 499 })
  })

  it('falls back to VAT on top when the backend sent no split', () => {
    expect(lineMoney(line({}))).toEqual({ exclTax: 1373, tax: 124, inclTax: 1497, unitInclTax: 499 })
  })

  it('takes the VAT out of a total that includes it', () => {
    expect(lineMoney(line({ quantity: 2, total_minor_units: 398, total_tax_minor_units: 33, total_includes_tax: true })))
      .toEqual({ exclTax: 365, tax: 33, inclTax: 398, unitInclTax: 199 })
  })

  it('prices an empty line at zero', () => {
    expect(lineMoney(line({ quantity: 0, total_minor_units: 0, total_tax_minor_units: 0 })).unitInclTax).toBe(0)
  })
})
