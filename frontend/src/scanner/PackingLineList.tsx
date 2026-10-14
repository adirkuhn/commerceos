import type { PickingSessionLineItem } from './types'
import { SCANNER_SOLD_EXPIRY_CLASS, ScanSessionLineRow } from './ScanSessionLines'
import { SoldExpiryNote } from '../oms/SoldExpiryNote'
import { usePackLineOrder, PACK_LINE_HIGHLIGHT_CLASS } from './usePackLineOrder'
import { useFlipList } from '../fulfillment/useFlipList'

interface PackingLineListProps {
  token: string
  sessionId: string
  lineItems: PickingSessionLineItem[]
  packedByLineItem: Map<string, number>
  readOnly: boolean
  onOpenLine: (lineItem: PickingSessionLineItem, packedSoFar: number) => void
}

// The scanner's packing lines: partly packed on top, then not packed yet,
// then done (see usePackLineOrder) -- same order and effects as the admin.
export function PackingLineList({ token, sessionId, lineItems, packedByLineItem, readOnly, onOpenLine }: PackingLineListProps) {
  const { lines, highlight } = usePackLineOrder(sessionId, lineItems, packedByLineItem)
  const flipRef = useFlipList(lines.map(li => li.id))
  return (
    <>
      {lines.map(li => {
        const packedQty = packedByLineItem.get(li.id) ?? 0
        const flash = highlight.get(li.id)
        return (
          <ScanSessionLineRow
            key={li.id} token={token} lineItem={li} doneQty={packedQty} disabled={readOnly}
            rowRef={flipRef(li.id)} highlightClass={flash && PACK_LINE_HIGHLIGHT_CLASS[flash]}
            onClick={() => onOpenLine(li, packedQty)}
          >
            <SoldExpiryNote date={li.sold_expiry_date} className={SCANNER_SOLD_EXPIRY_CLASS} />
          </ScanSessionLineRow>
        )
      })}
    </>
  )
}
