import { useAuth } from '../../auth/AuthContext'
import { useTaxClasses } from '../../tax/hooks/useTaxClasses'
import { useTaxRates } from '../../tax/hooks/useTaxRates'
import { useTenantSettings } from '../../settings/hooks/useTenantSettings'
import { resolveSkuTaxRate } from '../skuTax'

// Everything a SKU editor needs to show tax: the tenant's tax classes (for
// a picker), its settings (default currency/country, prices_include_tax),
// and a lookup from a SKU's tax class to the rate that applies.
export function useSkuTaxContext() {
  const { token } = useAuth()
  const { classes: taxClasses } = useTaxClasses(token ?? '')
  const { rates: taxRates } = useTaxRates(token ?? '')
  const { settings: tenantSettings } = useTenantSettings(token ?? '')
  const defaultTaxClass = taxClasses.find(c => c.is_default) ?? null
  const rateFor = (taxClassId: string | null | undefined) =>
    resolveSkuTaxRate(taxRates, taxClasses, taxClassId, tenantSettings?.default_tax_country)
  return { taxClasses, defaultTaxClass, tenantSettings, rateFor }
}
