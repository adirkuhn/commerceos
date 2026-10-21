import { useState } from 'react'
import i18next from 'i18next'
import * as api from './api'
import * as shippingApi from '../shipping/api'
import { useLabelConnection } from '../shipping/useLabelConnection'
import type { RecipientAddress, Shipment } from '../shipping/types'
import { loadShipmentLabelIntoTab, openShipmentLabel } from '../shipping/openLabel'
import { isAuthError } from '../auth/isAuthError'
import type { SalesOrder } from '../oms/types'
import { apiErrorText } from '../api/apiErrorText'

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
  const shipping = useLabelConnection(token)
  const { connection: shippingConnection, labelSize } = shipping
  const [labelStep, setLabelStep] = useState<LabelStep>(null)
  const [pendingRecipient, setPendingRecipient] = useState<RecipientAddress | null>(null)
  const [pendingCarrier, setPendingCarrier] = useState<number | null>(null)
  const [pendingPackageType, setPendingPackageType] = useState<number | null>(null)
  const [createdShipmentId, setCreatedShipmentId] = useState<string | null>(null)
  const [labelTab, setLabelTab] = useState<Window | null>(null)
  const [existingShipment, setExistingShipment] = useState<Shipment | null>(null)

  async function createLabel() {
    if (!shippingConnection) return
    // Before packing is finished, making the label is working the order --
    // take the lock first (the heartbeat claims it when free).
    if (!finished) {
      try {
        await api.touchPackingSession(token, sessionId)
        await reload()
      } catch (err) {
        if (isAuthError(err)) return onAuthError()
        setError(apiErrorText(err) ?? i18next.t('scanner:errors.someoneElsePacking'))
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
      setError(apiErrorText(err) ?? i18next.t('scanner:errors.loadLabel'))
    }
  }

  async function printLabel() {
    if (!shippingConnection || !pendingRecipient) return
    const tab = window.open('', '_blank')
    let shipment
    try {
      shipment = await shippingApi.createShipment(token, {
        carrier: shipping.carrierFor(pendingCarrier),
        package_type: pendingPackageType ?? shipping.defaultPackageType,
        recipient: pendingRecipient,
        connection_id: shippingConnection.id, sales_order_id: order.id,
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
      setError(apiErrorText(err) ?? i18next.t('scanner:errors.loadLabel'))
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
    shipping, shippingConnection, createdShipmentId, autoPrintEnabled: shipping.autoPrintEnabled,
    labelStep, setLabelStep, pendingRecipient, pendingCarrier, pendingPackageType,
    labelSize, setLabelSize: shipping.setLabelSize, labelTab, setLabelTab, existingShipment, setExistingShipment,
    createLabel, openLabel, printLabel, confirmAddress, reprintExisting, createNewOverExisting,
  }
}

export type PackingLabelFlow = ReturnType<typeof usePackingLabel>
