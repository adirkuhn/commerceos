import { useTranslation } from 'react-i18next'
import { guessCarrierFromShippingMethod } from '../shipping/carrierGuess'
import { AddressConfirmModal } from '../shipping/AddressConfirmModal'
import { LabelPreviewModal } from '../shipping/LabelPreviewModal'
import { ExistingShipmentConfirmModal } from '../shipping/ExistingShipmentConfirmModal'
import { LabelOpenedNotice } from '../shipping/LabelOpenedNotice'
import { LabelConnectionPicker } from '../shipping/ShippingProviderPicker'
import { providerLabel } from '../webhooks/providerLabel'
import type { SalesOrder } from '../oms/types'
import type { PackingLabelFlow } from './usePackingLabel'

interface PackingLabelModalsProps {
  token: string
  order: SalesOrder
  label: PackingLabelFlow
}

// Whichever step of the label flow is open: address, preview, the
// "label opened" notice, or the already-has-a-shipment confirm.
export function PackingLabelModals({ token, order, label }: PackingLabelModalsProps) {
  const { labelStep, pendingRecipient, pendingCarrier, pendingPackageType, labelTab, existingShipment, shipping } = label
  const { hasCarrierChoice } = shipping.capabilities
  return (
    <>
      {labelStep === 'address' && (
        <AddressConfirmModal
          token={token}
          initialAddress={order.shipping}
          fallbackEmail={order.billing?.email}
          connectionId={shipping.connection?.id}
          initialRecipient={pendingRecipient ?? undefined}
          initialCarrier={pendingCarrier ?? undefined}
          initialPackageType={pendingPackageType ?? undefined}
          defaultCarrier={hasCarrierChoice ? guessCarrierFromShippingMethod(order.shipping_method_title, order.shipping_method, shipping.defaultCarrier) : shipping.defaultCarrier}
          defaultPackageType={shipping.defaultPackageType}
          onNext={label.confirmAddress}
          onClose={() => label.setLabelStep(null)}
          providerPicker={<LabelConnectionPicker shipping={shipping} />}
          showCarrier={hasCarrierChoice}
        />
      )}
      {labelStep === 'preview' && pendingRecipient && (
        <LabelPreviewModal
          recipient={pendingRecipient}
          carrier={shipping.carrierFor(pendingCarrier)}
          packageType={pendingPackageType ?? shipping.defaultPackageType}
          providerName={shipping.connection ? providerLabel(shipping.connection.provider) : undefined}
          showCarrier={hasCarrierChoice}
          labelSize={label.labelSize}
          onLabelSizeChange={label.setLabelSize}
          onBack={() => label.setLabelStep('address')}
          onPrint={label.printLabel}
          onClose={() => label.setLabelStep(null)}
        />
      )}
      {labelTab && (
        <LabelOpenedNotice onPrint={() => labelTab.print()} onDismiss={() => label.setLabelTab(null)} />
      )}
      {existingShipment && (
        <ExistingShipmentConfirmModal
          token={token}
          shipment={existingShipment}
          onCancel={() => label.setExistingShipment(null)}
          onReprint={label.reprintExisting}
          onCreateNew={label.createNewOverExisting}
        />
      )}
    </>
  )
}

const labelButtonClass = 'w-full py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-200 text-sm font-medium hover:border-accent transition-colors'

// The label button shared by the review and done screens: create one, or
// print the one just made.
export function PackingLabelButton({ label }: { label: PackingLabelFlow }) {
  const { createdShipmentId } = label
  const { t } = useTranslation('scanner')
  return createdShipmentId ? (
    <button type="button" onClick={() => label.openLabel(createdShipmentId)} className={labelButtonClass}>
      {label.autoPrintEnabled ? t('label.retryPrint') : t('label.print')}
    </button>
  ) : label.shippingConnection ? (
    <button type="button" onClick={label.createLabel} className={labelButtonClass}>
      {t('label.create')}
    </button>
  ) : null
}
