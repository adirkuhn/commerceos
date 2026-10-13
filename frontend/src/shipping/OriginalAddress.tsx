import { useId } from 'react'
import { useTranslation } from 'react-i18next'
import type { Address } from '../oms/types'

// The order's shipping address exactly as the channel sent it, one line per
// field, shown above AddressConfirmModal's form. The form's fields are a
// best-effort split of this free text (see splitStreetAndNumber); keeping
// the untouched original in view lets staff spot a wrong split before a
// label goes out.
export function OriginalAddress({ address }: { address: Address }) {
  const { t } = useTranslation('shipping')
  const headingId = useId()
  const lines = [
    [address.first_name, address.last_name].filter(Boolean).join(' '),
    address.company,
    address.address_1,
    address.address_2,
    [address.postcode, address.city].filter(Boolean).join(' '),
    address.country,
  ].filter((line): line is string => Boolean(line))

  if (lines.length === 0) return null
  return (
    <div role="group" aria-labelledby={headingId} className="mb-4 rounded-lg border border-line bg-surface px-3 py-2">
      <p id={headingId} className="text-2xs font-medium text-ink-3 mb-1">{t('address.original')}</p>
      {lines.map((line, i) => <p key={i} className="text-xs text-ink">{line}</p>)}
    </div>
  )
}
