import { useEffect, useState } from 'react'
import * as shippingApi from './api'
import { useShippingConnections } from './useShippingConnections'
import { shippingProviderCapabilities } from './shippingProviders'

export type LabelSize = 'A4' | 'A6'

// Which shipping connection a label is made through, and that
// connection's label defaults. Starts on packing's default connection;
// staff can switch per label (selectConnection) before creating it.
// Shared by the three label flows (packing page, scanner packing, sales
// order page).
export function useLabelConnection(token: string) {
  const { connections, defaultConnection } = useShippingConnections(token)
  const [chosenId, setChosenId] = useState<string | null>(null)
  const connection = connections.find(c => c.id === chosenId) ?? defaultConnection
  const capabilities = shippingProviderCapabilities(connection?.provider)

  const [labelSize, setLabelSize] = useState<LabelSize>('A4')
  const [autoPrintEnabled, setAutoPrintEnabled] = useState(false)
  const [defaultCarrier, setDefaultCarrier] = useState(1)
  const [defaultPackageType, setDefaultPackageType] = useState(1)
  const connectionId = connection?.id
  useEffect(() => {
    if (!token || !connectionId) return
    let cancelled = false
    shippingApi.getShippingSettings(token, connectionId)
      .then(result => {
        if (cancelled) return
        setAutoPrintEnabled(result.settings?.auto_print_enabled ?? false)
        setLabelSize(result.settings?.default_label_size ?? 'A4')
        setDefaultCarrier(result.settings?.default_carrier ?? 1)
        setDefaultPackageType(result.settings?.default_package_type ?? 1)
      })
      // No settings saved yet is normal -- the label still works on the
      // built-in defaults.
      .catch(() => { if (!cancelled) setAutoPrintEnabled(false) })
    return () => { cancelled = true }
  }, [token, connectionId])

  // The carrier sent with the label: staff's pick when the provider has a
  // carrier choice, else the connection's own (a provider that is its own
  // carrier ignores a pick left over from another connection).
  function carrierFor(chosen: number | null | undefined): number {
    return capabilities.hasCarrierChoice ? chosen ?? defaultCarrier : defaultCarrier
  }

  return {
    connections, connection, capabilities, selectConnection: setChosenId,
    labelSize, setLabelSize, autoPrintEnabled, defaultCarrier, defaultPackageType, carrierFor,
  }
}

export type LabelConnection = ReturnType<typeof useLabelConnection>
