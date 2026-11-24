import { useTranslation } from 'react-i18next'
import { useShippingAddresses } from '../../customers/hooks/useShippingAddresses'
import { oneLineAddress, toOrderAddress } from '../../customers/shippingAddresses/addressLine'
import type { ShippingAddress } from '../../customers/types'
import type { Address } from '../types'

const SAME: (keyof Address)[] = ['first_name', 'last_name', 'company', 'address_1', 'address_2', 'city', 'postcode', 'country']
const matches = (a: ShippingAddress, v: Address) => SAME.every(k => (a[k as keyof ShippingAddress] ?? '') === (v[k] ?? ''))

// Picks which of the customer's shipping addresses the order goes to --
// shown only when they have more than one. value is the order's current
// shipping address: the matching book entry shows selected (the default,
// for a newly chosen customer), none for one typed by hand.
export function ShippingAddressPicker({ token, customerId, value, onPick }: {
  token: string
  customerId: string | null
  value: Address
  onPick: (a: Address) => void
}) {
  const { t } = useTranslation('customers')
  const { addresses } = useShippingAddresses(token, customerId)
  if (addresses.length < 2) return null
  const selected = addresses.find(a => matches(a, value))?.id ?? ''

  function pick(id: string) {
    const chosen = addresses.find(a => a.id === id)
    if (chosen) onPick(toOrderAddress(chosen))
  }

  return (
    <label className="block text-2xs font-medium text-ink-4 caps-label">
      {t('shippingAddresses.pick')}
      <select value={selected} onChange={e => pick(e.target.value)}
        className="mt-1 w-full px-2.5 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus normal-case">
        {selected === '' && <option value="">—</option>}
        {addresses.map(a => (
          <option key={a.id} value={a.id}>{[a.label, oneLineAddress(a)].filter(Boolean).join(' — ')}</option>
        ))}
      </select>
    </label>
  )
}
