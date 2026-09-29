import { useTranslation } from 'react-i18next'

interface ShippingLabelButtonProps {
  // Set once a label was made (or picked for reprint) in this visit.
  createdShipmentId: string | null
  autoPrintEnabled: boolean
  canCreate: boolean
  onOpen: (shipmentId: string) => void
  onCreate: () => void
  className: string
}

// The label button shared by the review and done screens: create one, or
// print the one just made.
export function ShippingLabelButton({ createdShipmentId, autoPrintEnabled, canCreate, onOpen, onCreate, className }: ShippingLabelButtonProps) {
  const { t } = useTranslation('packing')
  if (createdShipmentId) {
    return (
      <button type="button" onClick={() => onOpen(createdShipmentId)} className={className}>
        {autoPrintEnabled ? t('label.retryPrint') : t('label.print')}
      </button>
    )
  }
  if (!canCreate) return null
  return (
    <button type="button" onClick={onCreate} className={className}>
      {t('label.create')}
    </button>
  )
}
