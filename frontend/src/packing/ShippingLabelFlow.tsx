import type { SalesOrder } from '../oms/types'
import { guessCarrierFromShippingMethod } from '../shipping/carrierGuess'
import { AddressConfirmModal } from '../shipping/AddressConfirmModal'
import { LabelPreviewModal } from '../shipping/LabelPreviewModal'
import { ExistingShipmentConfirmModal } from '../shipping/ExistingShipmentConfirmModal'
import { LabelOpenedNotice } from '../shipping/LabelOpenedNotice'
import type { ShippingLabel } from './useShippingLabel'

interface ShippingLabelFlowProps {
  token: string | null
  activeOrder: SalesOrder | null
  label: ShippingLabel
}

// The label flow's modals, driven by useShippingLabel: address -> preview,
// the "label opened" notice, and the "label already exists" prompt.
export function ShippingLabelFlow({ token, activeOrder, label }: ShippingLabelFlowProps) {
  const { labelStep, pendingRecipient, pendingCarrier, pendingPackageType, connectionDefaultCarrier, connectionDefaultPackageType, labelTab, existingShipment } = label
  return (
    <>
      {labelStep === 'address' && activeOrder && (
        <AddressConfirmModal
          token={token ?? ''}
          initialAddress={activeOrder.shipping}
          fallbackEmail={activeOrder.billing?.email}
          connectionId={label.myParcelConnection?.id}
          initialRecipient={pendingRecipient ?? undefined}
          initialCarrier={pendingCarrier ?? undefined}
          initialPackageType={pendingPackageType ?? undefined}
          defaultCarrier={guessCarrierFromShippingMethod(activeOrder.shipping_method_title, activeOrder.shipping_method, connectionDefaultCarrier)}
          defaultPackageType={connectionDefaultPackageType}
          onNext={label.confirmAddress}
          onClose={label.closeFlow}
        />
      )}
      {labelStep === 'preview' && pendingRecipient && (
        <LabelPreviewModal
          recipient={pendingRecipient}
          carrier={pendingCarrier ?? connectionDefaultCarrier}
          packageType={pendingPackageType ?? connectionDefaultPackageType}
          labelSize={label.labelSize}
          onLabelSizeChange={label.setLabelSize}
          onBack={label.backToAddress}
          onPrint={label.printLabel}
          onClose={label.closeFlow}
        />
      )}
      {labelTab && (
        <LabelOpenedNotice onPrint={() => labelTab.print()} onDismiss={label.dismissLabelTab} />
      )}
      {existingShipment && (
        <ExistingShipmentConfirmModal
          token={token ?? ''}
          shipment={existingShipment}
          onCancel={label.cancelExisting}
          onReprint={label.reprintExisting}
          onCreateNew={label.createNewInsteadOfExisting}
        />
      )}
    </>
  )
}
