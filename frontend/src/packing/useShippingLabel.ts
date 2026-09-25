import { useState, useEffect } from 'react'
import type { SalesOrder } from '../oms/types'
import { useConnections } from '../integrations/hooks/useConnections'
import * as shippingApi from '../shipping/api'
import type { RecipientAddress, Shipment } from '../shipping/types'
import { loadShipmentLabelIntoTab, openShipmentLabel } from '../shipping/openLabel'

export type LabelStep = 'address' | 'preview'

interface ShippingLabelOptions {
  token: string | null
  activeOrder: SalesOrder | null
  // Runs before a new label is made; false aborts (the caller shows why).
  beforeCreate: () => Promise<boolean>
  onError: (message: string) => void
}

export interface AddressChoice {
  carrier: number
  packageType: number
  recipient: RecipientAddress
}

// The 3-step MyParcel label flow on the Packing page (confirm address ->
// preview -> print), plus reprinting a label the order already has. Same
// flow as Picking.tsx used to have -- moved here now that packing (not
// picking) is what gates "ready to ship".
export function useShippingLabel({ token, activeOrder, beforeCreate, onError }: ShippingLabelOptions) {
  const { connections } = useConnections(token ?? '')
  const myParcelConnection = connections.find(c => c.provider === 'myparcel')
  const [labelStep, setLabelStep] = useState<LabelStep | null>(null)
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
    if (!token || !myParcelConnection) return
    shippingApi.getShippingSettings(token, myParcelConnection.id)
      .then(result => {
        setAutoPrintEnabled(result.settings?.auto_print_enabled ?? false)
        setLabelSize(result.settings?.default_label_size ?? 'A4')
        setConnectionDefaultCarrier(result.settings?.default_carrier ?? 1)
        setConnectionDefaultPackageType(result.settings?.default_package_type ?? 1)
      })
      .catch(() => setAutoPrintEnabled(false))
  }, [token, myParcelConnection])

  useEffect(() => {
    setLabelStep(null)
    setPendingRecipient(null)
    setPendingCarrier(null)
    setPendingPackageType(null)
    setCreatedShipmentId(null)
    setLabelTab(null)
    setExistingShipment(null)
  }, [activeOrder?.id])

  async function createLabel() {
    if (!token || !activeOrder) return
    if (!await beforeCreate()) return
    const shipments = await shippingApi.listShipmentsForOrder(token, activeOrder.id)
    const active = shipments.find(s => !s.cancelled_at)
    if (active) {
      setExistingShipment(active)
      return
    }
    setLabelStep('address')
  }

  async function openLabel(shipmentId: string) {
    if (!token) return
    try {
      setLabelTab(await openShipmentLabel(token, shipmentId, labelSize))
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Failed to load label')
    }
  }

  async function printLabel() {
    if (!token || !activeOrder || !myParcelConnection || !pendingRecipient) return
    const tab = window.open('', '_blank')
    let shipment
    try {
      shipment = await shippingApi.createShipment(token, {
        carrier: pendingCarrier ?? connectionDefaultCarrier,
        package_type: pendingPackageType ?? connectionDefaultPackageType,
        recipient: pendingRecipient,
        connection_id: myParcelConnection.id, sales_order_id: activeOrder.id,
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
      onError(err instanceof Error ? err.message : 'Failed to load label')
    }
  }

  function confirmAddress({ carrier, packageType, recipient }: AddressChoice) {
    setPendingCarrier(carrier)
    setPendingPackageType(packageType)
    setPendingRecipient(recipient)
    setLabelStep('preview')
  }

  function reprintExisting() {
    if (!existingShipment) return
    setCreatedShipmentId(existingShipment.id)
    openLabel(existingShipment.id)
    setExistingShipment(null)
  }

  function createNewInsteadOfExisting() {
    setExistingShipment(null)
    setLabelStep('address')
  }

  // The order left the screen -- drop any half-done label flow.
  function clear() {
    setCreatedShipmentId(null)
    setLabelStep(null)
    setPendingRecipient(null)
    setExistingShipment(null)
  }

  return {
    myParcelConnection, autoPrintEnabled, createdShipmentId,
    labelStep, pendingRecipient, pendingCarrier, pendingPackageType,
    connectionDefaultCarrier, connectionDefaultPackageType,
    labelSize, setLabelSize, labelTab, existingShipment,
    createLabel, openLabel, printLabel, confirmAddress,
    closeFlow: () => setLabelStep(null), backToAddress: () => setLabelStep('address'),
    dismissLabelTab: () => setLabelTab(null),
    cancelExisting: () => setExistingShipment(null), reprintExisting, createNewInsteadOfExisting,
    clear,
  }
}

export type ShippingLabel = ReturnType<typeof useShippingLabel>
