import { AlertTriangle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { defaultVatProblem } from '../tax/defaultVatClass'
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
  const problem = defaultVatProblem(classes, rates, country)
  if (!problem) return null
  const def = classes.find(c => c.is_default)
  return (
    <div role="alert" className="flex gap-2 rounded-xl border border-warning/40 bg-warning/5 p-3 text-xs text-ink-2">
      <AlertTriangle size={14} className="text-warning shrink-0 mt-0.5" aria-hidden />
      <p>
        {problem === 'noDefault'
          ? t('tax.noDefaultClass')
          : t('tax.defaultHasNoRate', { name: def?.name ?? '', country: country.toUpperCase() })}
      </p>
    </div>
  )
}
