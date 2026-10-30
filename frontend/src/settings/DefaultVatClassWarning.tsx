import { AlertTriangle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { taxSetupGap } from '../tax/taxSetupGap'
import type { TaxClass, TaxRate } from '../tax/types'

interface Props {
  classes: TaxClass[]
  rates: TaxRate[]
  country: string
}

// Shown on Settings -> Tax while products without their own VAT class get
// no VAT rate -- they'd be sold at 0% without anyone noticing.
export function DefaultVatClassWarning({ classes, rates, country }: Props) {
  const { t } = useTranslation('settings')
  const gap = taxSetupGap(classes, rates, country)
  if (!gap) return null
  const def = classes.find(c => c.is_default)
  const text = {
    country: t('tax.noTaxCountry'),
    defaultClass: t('tax.noDefaultClass'),
    rate: t('tax.defaultHasNoRate', { name: def?.name ?? '', country: country.toUpperCase() }),
  }[gap]
  return (
    <div role="alert" className="flex gap-2 rounded-xl border border-warning/40 bg-warning/5 p-3 text-xs text-ink-2">
      <AlertTriangle size={14} className="text-warning shrink-0 mt-0.5" aria-hidden />
      <p>{text}</p>
    </div>
  )
}
