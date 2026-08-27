import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import * as shippingApi from '../../shipping/api'

interface ShippingLabelPrintViewProps {
  token: string
  shipmentId: string
  // Physical page format to fetch/print -- a retrieval-time choice (see
  // shipping/api.ts's getShipmentLabel), so it's plain component state here,
  // re-fetchable via the size toggle below without recreating the shipment.
  defaultSize?: 'A4' | 'A6'
  onClose: () => void
}

// Fetches the label as a blob and renders it full-screen so window.print()
// prints the actual label content -- same trigger-on-mount pattern as
// PickingListPrintView/InvoicePrintView (window.print() in a useEffect,
// print:hidden close button), adapted for a binary PDF response instead of
// plain HTML: an <embed> pointed at a blob: URL is how a PDF gets both
// displayed AND included in the browser's print output, since window.print()
// only prints what's actually rendered in the page, not the source bytes.
// Firing print on the embed's onLoad (not on component mount) matters --
// the PDF hasn't rendered yet at mount time, and printing before it has
// loaded produces a blank page.
//
// ZPL-format labels aren't visually renderable (it's a raw thermal-printer
// command language, not a document format) -- this view only knows how to
// preview/print PDF; a ZPL label gets a plain download link instead of a
// broken embed + garbage print output.
export function ShippingLabelPrintView({ token, shipmentId, defaultSize, onClose }: ShippingLabelPrintViewProps) {
  const [size, setSize] = useState<'A4' | 'A6'>(defaultSize ?? 'A4')
  const [blobUrl, setBlobUrl] = useState<string | null>(null)
  const [contentType, setContentType] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    let url: string | null = null
    // Reset to the loading state on every size change too, not just mount --
    // switching A4<->A6 re-fetches (size is retrieval-time, no new shipment
    // needed) and the stale previous-size embed shouldn't linger mid-fetch.
    setBlobUrl(null)
    setError('')
    shippingApi.getShipmentLabel(token, shipmentId, size)
      .then(blob => {
        url = URL.createObjectURL(blob)
        setContentType(blob.type)
        setBlobUrl(url)
      })
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to load label'))
    return () => {
      if (url) URL.revokeObjectURL(url)
    }
  }, [token, shipmentId, size])

  const isPdf = contentType.includes('pdf')

  return (
    <div className="fixed inset-0 z-50 bg-white flex flex-col">
      <div className="print:hidden fixed top-4 right-4 flex items-center gap-2 z-10">
        <div className="flex items-center rounded-lg border border-gray-200 overflow-hidden bg-white shadow-sm">
          {(['A4', 'A6'] as const).map(s => (
            <button
              key={s}
              type="button"
              onClick={() => setSize(s)}
              aria-pressed={size === s}
              className={`px-3 py-2 text-xs font-medium ${size === s ? 'bg-blue-600 text-white' : 'bg-white text-gray-600 hover:bg-gray-50'}`}
            >
              {s}
            </button>
          ))}
        </div>
        <button
          onClick={onClose}
          className="w-9 h-9 flex items-center justify-center rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-600"
          title="Close"
        >
          <X size={16} />
        </button>
      </div>

      {error && <p className="m-auto text-sm text-red-500">{error}</p>}

      {!error && !blobUrl && <p className="m-auto text-sm text-gray-500">Loading label…</p>}

      {!error && blobUrl && isPdf && (
        <embed
          src={blobUrl}
          type="application/pdf"
          className="flex-1 w-full"
          onLoad={() => window.print()}
        />
      )}

      {!error && blobUrl && !isPdf && (
        <div className="m-auto text-center text-sm text-gray-600 space-y-3">
          <p>This label is in ZPL (thermal printer) format -- there's no visual preview for it.</p>
          <a href={blobUrl} download={`shipping-label-${shipmentId}.zpl`} className="text-blue-600 underline">
            Download the ZPL file
          </a>
        </div>
      )}
    </div>
  )
}
