import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

// Names the products on an order or quote whose VAT rate isn't configured
// (no rate for their VAT class, or no default class) -- their lines are at
// a 0% nobody chose.
export function MissingVatWarning({ products }: { products: string[] }) {
  const { t } = useTranslation('settings')
  if (products.length === 0) return null
  return (
    <p role="status" className="text-xs text-ink-2 bg-warning/10 rounded-lg px-3 py-2">
      {t('tax.missingVat', { products: products.join(', ') })}{' '}
      <Link to="/settings/tax" className="underline">{t('tax.missingVatFix')}</Link>
    </p>
  )
}
