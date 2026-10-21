import { useEffect, useRef } from 'react'
import { useLocation } from 'react-router-dom'
import { MapPin } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useShippingConnections, type UseShippingConnections } from '../useShippingConnections'
import { missingSetupTexts } from '../connectionReadiness'
import { SHIPPING_ADDRESSES_ANCHOR } from '../shippingSetupPaths'
import type { UpdateShippingAddressesInput } from '../types'
import { useShippingAddresses } from './useShippingAddresses'
import { ShippingAddressesForm } from './ShippingAddressesForm'

interface ShippingAddressesSettingProps {
  token: string
  // Owners/admins change it; everyone else sees it read-only (the backend
  // enforces this too).
  canEdit: boolean
}

// "Shipping addresses": the tenant's label sender and return address.
// Shown once there is a shipping connection; lists what a connection still
// misses (re-checked after saving) and scrolls into view when the URL
// carries its anchor (label flows link here).
export function ShippingAddressesSetting({ token, canEdit }: ShippingAddressesSettingProps) {
  const shipping = useShippingConnections(token)
  if (shipping.loading || shipping.connections.length === 0) return null
  return <ShippingAddressesPanel token={token} canEdit={canEdit} shipping={shipping} />
}

function ShippingAddressesPanel({ token, canEdit, shipping }: ShippingAddressesSettingProps & { shipping: UseShippingConnections }) {
  const { t } = useTranslation('shipping')
  const { connections, refresh } = shipping
  const { addresses, loading, error, save } = useShippingAddresses(token)
  const location = useLocation()
  const sectionRef = useRef<HTMLElement>(null)
  const focused = location.hash === `#${SHIPPING_ADDRESSES_ANCHOR}`

  useEffect(() => {
    if (!focused) return
    // jsdom has no layout engine; scrolling is a convenience anyway.
    try { sectionRef.current?.scrollIntoView({ block: 'start' }) } catch { /* no-op */ }
  }, [focused])

  async function handleSave(input: UpdateShippingAddressesInput) {
    await save(input)
    await refresh()
  }

  const warnings = connections.flatMap(c => missingSetupTexts(t, c))
  return (
    <section
      id={SHIPPING_ADDRESSES_ANCHOR} ref={sectionRef} aria-labelledby="shipping-addresses-title"
      className="flex items-start gap-3 px-4 py-3 rounded-xl bg-raised border border-line scroll-mt-4"
    >
      <div className="w-8 h-8 rounded-lg bg-accent/10 flex items-center justify-center shrink-0">
        <MapPin size={15} className="text-accent" />
      </div>
      <div className="flex-1 min-w-0 max-w-2xl">
        <h2 id="shipping-addresses-title" className="text-xs font-semibold text-ink">{t('addresses.title')}</h2>
        <p className="text-2xs text-ink-4 mb-2">{t('addresses.body')}</p>
        {warnings.map(w => <p key={w} className="text-xs text-warning mb-2">{w}</p>)}
        {loading && <p className="text-2xs text-ink-4">{t('addresses.loading')}</p>}
        {error && <p role="alert" className="text-xs text-danger">{error}</p>}
        {addresses && <ShippingAddressesForm initial={addresses} canEdit={canEdit} onSave={handleSave} />}
      </div>
    </section>
  )
}
