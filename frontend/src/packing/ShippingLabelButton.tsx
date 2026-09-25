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
  if (createdShipmentId) {
    return (
      <button type="button" onClick={() => onOpen(createdShipmentId)} className={className}>
        {autoPrintEnabled ? 'Retry print' : 'Print shipping label'}
      </button>
    )
  }
  if (!canCreate) return null
  return (
    <button type="button" onClick={onCreate} className={className}>
      Create shipping label
    </button>
  )
}
