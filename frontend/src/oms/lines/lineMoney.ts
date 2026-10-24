import type { SalesOrderLineItem } from '../types'

export interface LineMoney {
  exclTax: number
  tax: number
  inclTax: number
  unitInclTax: number
}

// A line's money without VAT, its VAT and with VAT. The backend sends the
// split; a response without it is read the way Woo stores lines (VAT on top)
// unless the line says its total already includes the VAT.
export function lineMoney(li: SalesOrderLineItem): LineMoney {
  const tax = li.total_tax_minor_units
  const exclTax = li.total_excl_tax_minor_units ?? (li.total_includes_tax ? li.total_minor_units - tax : li.total_minor_units)
  const inclTax = li.total_incl_tax_minor_units ?? exclTax + tax
  return { exclTax, tax, inclTax, unitInclTax: li.quantity > 0 ? Math.round(inclTax / li.quantity) : 0 }
}
