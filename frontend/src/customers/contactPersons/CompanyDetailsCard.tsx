import { ExternalLink, MapPin } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Customer } from '../types'

// mapsQuery is the billing address on one line, or '' when there is none.
function mapsQuery(c: Customer): string {
  const a = c.billing ?? {}
  const town = [a.postcode, a.city].filter(Boolean).join(' ')
  return [a.address_1, town, a.country].filter(Boolean).join(', ')
}

// A customer's business details: client number, KVK number (linked to the
// Dutch Chamber of Commerce), VAT ID, and the address on a map.
export function CompanyDetailsCard({ customer }: { customer: Customer }) {
  const { t } = useTranslation('customers')
  const query = mapsQuery(customer)
  const row = (label: string, value: React.ReactNode) => (
    <div>
      <dt className="text-2xs font-semibold caps-label text-ink-4">{label}</dt>
      <dd className="text-ink">{value}</dd>
    </div>
  )

  return (
    <div className="rounded-xl border border-line p-4 space-y-3 text-xs">
      <div className="flex items-center justify-between">
        <h3 className="text-xs font-semibold text-ink">{t('company.title')}</h3>
        {query && (
          <a href={`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(query)}`} target="_blank" rel="noopener noreferrer"
            aria-label={t('company.openMap')} title={t('company.openMap')}
            className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover text-ink-3"><MapPin size={12} /></a>
        )}
      </div>
      <dl className="space-y-2">
        {customer.client_number && row(t('identity.clientNumber'), <span className="tabular-nums">{customer.client_number}</span>)}
        {customer.kvk_number && row(t('identity.kvkNumber'), (
          <a href={`https://www.kvk.nl/bestellen/#/${encodeURIComponent(customer.kvk_number)}`} target="_blank" rel="noopener noreferrer"
            className="inline-flex items-center gap-1 text-accent hover:underline tabular-nums">
            {customer.kvk_number} <ExternalLink size={10} aria-hidden />
          </a>
        ))}
        {customer.vat_number && row(t('identity.vatNumber'), customer.vat_number)}
      </dl>
    </div>
  )
}
