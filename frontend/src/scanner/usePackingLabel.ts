import { useEffect, useState } from 'react'
import * as api from './api'
import { useConnections } from '../integrations/hooks/useConnections'
import * as shippingApi from '../shipping/api'
import type { RecipientAddress, Shipment } from '../shipping/types'
import { loadShipmentLabelIntoTab, openShipmentLabel } from '../shipping/openLabel'
import { isAuthError } from '../auth/isAuthError'
import type { SalesOrder } from '../oms/types'

interface PackingLabelOptions {
  token: string
  sessionId: string
  order: SalesOrder
  // Packing already finished -- the label no longer needs the lock.
  finished: boolean
  reload: () => Promise<void>
  onAuthError: () => void
  setError: (message: string) => void
}

export type LabelStep = 'address' | 'preview' | null

// Same 3-step label flow as PickingScreen used to have -- see Picking.tsx's
// doc comment; moved here now that packing (not picking) gates shipping.
// The label flow runs both before finishing (from the working screen) and
// after (from the done screen).
export function usePackingLabel({ token, sessionId, order, finished, reload, onAuthError, setError }: PackingLabelOptions) {
  const { connections } = useConnections(token)
  const myParcelConnection = connections.find(c => c.provider === 'myparcel')
  const [labelStep, setLabelStep] = useState<LabelStep>(null)
  const [pendingRecipient, setPendingRecipient] = useState<RecipientAddress | null>(null)
  const [pendingCarrier, setPendingCarrier] = useState<number | null>(null)
  const [pendingPackageType, setPendingPackageType] = useState<number | null>(null)
  const [labelSize, setLabelSize] = useState<'A4' | 'A6'>('A4')
  const [createdShipmentId, setCreatedShipmentId] = useState<string | null>(null)
  const [labelTab, setLabelTab] = useState<Window | null>(null)
  const [existingShipment, setExistingShipment] = useState<Shipment | null>(null)
  const [autoPrintEnabled, setAutoPrintEnabled] = useState(false)
  const [connectionDefaultCarrier, setConnectionDefaultCarrier] = useState(1)
  const [connectionDefaultPackageType, setConnectionDefaultPackageType] = useState(1)
  useEffect(() => {
    if (!myParcelConnection) return
    shippingApi.getShippingSettings(token, myParcelConnection.id)
      .then(result => {
        setAutoPrintEnabled(result.settings?.auto_print_enabled ?? false)
        setLabelSize(result.settings?.default_label_size ?? 'A4')
        setConnectionDefaultCarrier(result.settings?.default_carrier ?? 1)
        setConnectionDefaultPackageType(result.settings?.default_package_type ?? 1)
      })
      .catch(() => setAutoPrintEnabled(false))
  }, [token, myParcelConnection])

  async function createLabel() {
    if (!myParcelConnection) return
    // Before packing is finished, making the label is working the order --
    // take the lock first (the heartbeat claims it when free).
    if (!finished) {
      try {
        await api.touchPackingSession(token, sessionId)
        await reload()
      } catch (err) {
        if (isAuthError(err)) return onAuthError()
        setError(err instanceof Error ? err.message : 'Someone else is packing this order')
        return
      }
    }
    const shipments = await shippingApi.listShipmentsForOrder(token, order.id)
    const active = shipments.find(s => !s.cancelled_at)
    if (active) {
      setExistingShipment(active)
      return
    }
    setLabelStep('address')
  }

  async function openLabel(shipmentId: string) {
    try {
      setLabelTab(await openShipmentLabel(token, shipmentId, labelSize))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load label')
    }
  }

  async function printLabel() {
    if (!myParcelConnection || !pendingRecipient) return
    const tab = window.open('', '_blank')
    let shipment
    try {
      shipment = await shippingApi.createShipment(token, {
        carrier: pendingCarrier ?? connectionDefaultCarrier,
        package_type: pendingPackageType ?? connectionDefaultPackageType,
        recipient: pendingRecipient,
        connection_id: myParcelConnection.id, sales_order_id: order.id,
      })
    } catch (err) {
      tab?.close()
      throw err
    }
    setLabelStep(null)
    setCreatedShipmentId(shipment.id)
    try {
      setLabelTab(await loadShipmentLabelIntoTab(tab, token, shipment.id, labelSize))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load label')
    }
  }

  function confirmAddress(next: { carrier: number; packageType: number; recipient: RecipientAddress }) {
    setPendingCarrier(next.carrier)
    setPendingPackageType(next.packageType)
    setPendingRecipient(next.recipient)
    setLabelStep('preview')
  }

  function reprintExisting() {
    if (!existingShipment) return
    setCreatedShipmentId(existingShipment.id)
    openLabel(existingShipment.id)
    setExistingShipment(null)
  }

  function createNewOverExisting() {
    setExistingShipment(null)
    setLabelStep('address')
  }

  return {
    myParcelConnection, createdShipmentId, autoPrintEnabled,
    labelStep, setLabelStep, pendingRecipient, pendingCarrier, pendingPackageType,
    labelSize, setLabelSize, labelTab, setLabelTab, existingShipment, setExistingShipment,
    connectionDefaultCarrier, connectionDefaultPackageType,
    createLabel, openLabel, printLabel, confirmAddress, reprintExisting, createNewOverExisting,
  }
}

export type PackingLabelFlow = ReturnType<typeof usePackingLabel>
