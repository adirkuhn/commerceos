import { useTranslation } from 'react-i18next'
import type { PickingSessionLineItem } from './types'
import { SCANNER_SOLD_EXPIRY_CLASS, ScanSessionLineRow } from './ScanSessionLines'
import { SoldExpiryNote } from '../oms/SoldExpiryNote'
import { usePackLineOrder, PACK_LINE_HIGHLIGHT_CLASS } from './usePackLineOrder'
import { useFlipList } from '../fulfillment/useFlipList'

interface PickingLineListProps {
  token: string
  // The picking session -- opening another order sorts without effects.
  sessionId: string
  lineItems: PickingSessionLineItem[]
  pickedByLineItem: Map<string, number>
  readOnly: boolean
  onOpenLine: (lineItem: PickingSessionLineItem, pickedSoFar: number) => void
}

// The scanner's picking lines: the line just picked on top, then partly
// picked, then not picked yet, then done (see usePackLineOrder) -- same
// order and effects as packing and the admin.
export function PickingLineList({ token, sessionId, lineItems, pickedByLineItem, readOnly, onOpenLine }: PickingLineListProps) {
  const { t } = useTranslation('scanner')
  const { lines, highlight } = usePackLineOrder(sessionId, lineItems, pickedByLineItem)
  const flipRef = useFlipList(lines.map(li => li.id))
  return (
    <>
      {lines.map(li => {
        const pickedQty = pickedByLineItem.get(li.id) ?? 0
        const flash = highlight.get(li.id)
        return (
          <ScanSessionLineRow
            key={li.id} token={token} lineItem={li} doneQty={pickedQty} disabled={readOnly}
            rowRef={flipRef(li.id)} highlightClass={flash && PACK_LINE_HIGHLIGHT_CLASS[flash]}
            onClick={() => onOpenLine(li, pickedQty)}
          >
            <SoldExpiryNote date={li.sold_expiry_date} className={SCANNER_SOLD_EXPIRY_CLASS} />
            {!li.sku_id && <p className="text-xs text-amber-400">{t('shared.notLinked')}</p>}
          </ScanSessionLineRow>
        )
      })}
    </>
  )
}
