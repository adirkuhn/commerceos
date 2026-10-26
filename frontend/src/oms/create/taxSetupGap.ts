import type { TaxClass, TaxRate } from '../../tax/types'

export type TaxSetupGap = 'country' | 'defaultClass' | 'rate'

// taxSetupGap is what Settings -> Tax still lacks for the order form to
// find VAT for a product on the default class (and for shipping): the
// default tax country, a default class, or that class's rate in the
// country. Null when it is all there.
export function taxSetupGap(classes: TaxClass[], rates: TaxRate[], country: string | null | undefined): TaxSetupGap | null {
  if (!country) return 'country'
  const defaultClass = classes.find(c => c.is_default)
  if (!defaultClass) return 'defaultClass'
  const hasRate = rates.some(r => r.tax_class_id === defaultClass.id && r.country.toUpperCase() === country.toUpperCase())
  return hasRate ? null : 'rate'
}
