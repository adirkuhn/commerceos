import { Link } from 'react-router-dom'
import { AlertTriangle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { ShippingConnection } from './types'
import { shippingConnectionLabel } from './shippingProviders'
import { connectionReady, missingSetupTexts } from './connectionReadiness'
import { SHIPPING_ADDRESSES_PATH } from './shippingSetupPaths'
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

// Why the selected connection can't create labels yet, with a link to fix
// it; nothing when it is ready.
export function ConnectionNotReadyNotice({ connection }: { connection: ShippingConnection | null }) {
  const { t } = useTranslation('shipping')
  if (!connection || connectionReady(connection)) return null
  return (
    <div role="alert" className="flex items-start gap-2 mt-2 px-2.5 py-2 rounded-lg bg-amber-50 dark:bg-amber-500/10 text-xs text-warning">
      <AlertTriangle size={14} className="shrink-0 mt-0.5" />
      <div>
        {missingSetupTexts(t, connection).map(text => <p key={text}>{text}</p>)}
        <Link to={SHIPPING_ADDRESSES_PATH} className="font-medium underline-offset-2 hover:underline">{t('readiness.fixLink')}</Link>
      </div>
    </div>
  )
}

// The picker wired to a label flow's useLabelConnection, plus the warning
// when the selected connection isn't ready.
export function LabelConnectionPicker({ shipping }: { shipping: LabelConnection }) {
  return (
    <>
      <ShippingProviderPicker connections={shipping.connections} selectedId={shipping.connection?.id} onSelect={shipping.selectConnection} />
      <ConnectionNotReadyNotice connection={shipping.connection} />
    </>
  )
}
