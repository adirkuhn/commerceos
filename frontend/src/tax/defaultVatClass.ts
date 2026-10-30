import type { TaxClass, TaxRate } from './types'

export type DefaultVatProblem = 'noDefault' | 'noRateForCountry'

// Why products without their own VAT class would get no VAT rate (the
// backend's domain/tax.RateFor): no default class, or the default class has
// no rate for the shop's tax country. null when they get one -- a 0% rate
// that is configured counts as a rate.
export function defaultVatProblem(classes: TaxClass[], rates: TaxRate[], country: string): DefaultVatProblem | null {
  const def = classes.find(c => c.is_default)
  if (!def) return 'noDefault'
  const hasRate = rates.some(r => r.tax_class_id === def.id && r.country.toUpperCase() === country.toUpperCase())
  return hasRate ? null : 'noRateForCountry'
}
