import { useTranslation } from 'react-i18next'
import type { ShippingConnection } from './types'
import { shippingConnectionLabel } from './shippingProviders'
import type { LabelConnection } from './useLabelConnection'

interface ShippingProviderPickerProps {
  connections: readonly ShippingConnection[]
  selectedId: string | undefined
  onSelect: (connectionId: string) => void
}

// Which shipping provider this label goes through. A dropdown when the
// tenant has several shipping connections; with just one there is nothing
// to choose, so it only names the provider (staff still see who ships it).
export function ShippingProviderPicker({ connections, selectedId, onSelect }: ShippingProviderPickerProps) {
  const { t } = useTranslation('shipping')
  if (connections.length === 0) return null
  if (connections.length === 1) {
    return (
      <p className="text-2xs text-ink-3">
        {t('provider.via', { provider: shippingConnectionLabel(connections[0], connections) })}
      </p>
    )
  }
  return (
    <div>
      <label htmlFor="ship-provider" className="block text-2xs font-medium text-ink-3 mb-1">{t('provider.label')}</label>
      <select
        id="ship-provider" value={selectedId ?? ''} onChange={e => onSelect(e.target.value)}
        className="w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink"
      >
        {connections.map(c => <option key={c.id} value={c.id}>{shippingConnectionLabel(c, connections)}</option>)}
      </select>
    </div>
  )
}

// The picker wired to a label flow's useLabelConnection.
export function LabelConnectionPicker({ shipping }: { shipping: LabelConnection }) {
  return <ShippingProviderPicker connections={shipping.connections} selectedId={shipping.connection?.id} onSelect={shipping.selectConnection} />
}
