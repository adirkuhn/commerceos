import { useEffect, useRef, useState } from 'react'
import { ArrowLeft, Check, PackageCheck, ChevronDown, ChevronUp, Camera } from 'lucide-react'
import * as api from './api'
import { CameraScanner } from './CameraScanner'
import type { BookingDetail, CompleteBookingResult } from './types'

interface BookingScreenProps {
  token: string
  bookingId: string
  onExit: () => void
  onAuthError: () => void
}

function isAuthError(err: unknown): boolean {
  return err instanceof Error && /401|unauthorized/i.test(err.message)
}

export function BookingScreen({ token, bookingId, onExit, onAuthError }: BookingScreenProps) {
  const [detail, setDetail] = useState<BookingDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [code, setCode] = useState('')
  const [quantity, setQuantity] = useState('1')
  const [lotNumber, setLotNumber] = useState('')
  const [expiryDate, setExpiryDate] = useState('')
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [scanning, setScanning] = useState(false)
  const [cameraOpen, setCameraOpen] = useState(false)
  const [finishing, setFinishing] = useState(false)
  const [result, setResult] = useState<CompleteBookingResult | null>(null)
  const codeInputRef = useRef<HTMLInputElement>(null)

  const load = async () => {
    try {
      setDetail(await api.getBooking(token, bookingId))
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to load booking')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [bookingId])
  useEffect(() => { codeInputRef.current?.focus() }, [detail])

  async function submitScan(scanCode: string) {
    if (!scanCode.trim()) return
    setError('')
    setScanning(true)
    try {
      await api.scanBookingItem(token, bookingId, {
        code: scanCode.trim(), quantity: parseInt(quantity, 10) || 1,
        lot_number: lotNumber || undefined, expiry_date: expiryDate || null,
      })
      setCode('')
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Scan failed')
    } finally {
      setScanning(false)
      codeInputRef.current?.focus()
    }
  }

  function handleScan(e: React.FormEvent) {
    e.preventDefault()
    submitScan(code)
  }

  function handleCameraDecode(text: string) {
    setCameraOpen(false)
    setCode(text)
    submitScan(text)
  }

  async function handlePause() {
    try {
      await api.pauseBooking(token, bookingId)
      onExit()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to pause')
    }
  }

  async function handleFinish() {
    setFinishing(true)
    try {
      setResult(await api.completeBooking(token, bookingId))
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to complete')
    } finally {
      setFinishing(false)
    }
  }

  if (result) {
    return (
      <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100 p-6">
        <div className="flex-1 flex flex-col items-center justify-center text-center gap-3">
          <div className="w-14 h-14 rounded-2xl bg-green-600/20 flex items-center justify-center">
            <PackageCheck size={26} className="text-green-400" />
          </div>
          <h1 className="text-lg font-semibold">Booking complete</h1>
          {result.missing_items.length === 0 ? (
            <p className="text-sm text-gray-400">Every line was fully scanned.</p>
          ) : (
            <div className="w-full max-w-sm mt-2 space-y-2 text-left">
              <p className="text-sm text-amber-400">{result.missing_items.length} line(s) weren't fully scanned:</p>
              {result.missing_items.map(m => (
                <div key={m.line_item_id} className="rounded-lg bg-[#1F2937] border border-[#2D3748] px-3 py-2">
                  <p className="text-sm text-gray-200">{m.raw_description}</p>
                  <p className="text-xs text-gray-500">expected {m.expected_qty} · scanned {m.scanned_qty}</p>
                </div>
              ))}
            </div>
          )}
        </div>
        <button onClick={onExit} className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold hover:bg-blue-700">
          Done
        </button>
      </div>
    )
  }

  const scannedBySKU = new Map<string, number>()
  for (const s of detail?.scans ?? []) {
    scannedBySKU.set(s.sku_id, (scannedBySKU.get(s.sku_id) ?? 0) + s.quantity)
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
        <button onClick={handlePause} className="p-1 -ml-1" title="Pause and go back">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Booking</h1>
        <button
          onClick={handleFinish}
          disabled={finishing}
          className="px-3 py-1.5 rounded-lg bg-green-600 text-white text-xs font-semibold hover:bg-green-700 disabled:opacity-50"
        >
          {finishing ? 'Finishing…' : 'Finish'}
        </button>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
        {loading && <p className="text-center text-sm text-gray-500 py-8">Loading…</p>}
        {detail?.line_items.map(li => {
          const scannedQty = li.sku_id ? scannedBySKU.get(li.sku_id) ?? 0 : 0
          const done = scannedQty >= li.quantity
          return (
            <div key={li.id} className="flex items-center gap-3 rounded-xl bg-[#1F2937] border border-[#2D3748] px-3.5 py-3">
              {done && <Check size={16} className="text-green-400 shrink-0" />}
              <div className="flex-1 min-w-0">
                <p className="text-sm truncate">{li.raw_description}</p>
                {!li.sku_id && <p className="text-xs text-amber-400">Not linked to a product yet</p>}
              </div>
              <p className={`text-sm font-medium shrink-0 ${done ? 'text-green-400' : 'text-gray-300'}`}>
                {scannedQty} / {li.quantity}
              </p>
            </div>
          )
        })}
      </div>

      <form onSubmit={handleScan} className="border-t border-[#2D3748] p-4 space-y-2 shrink-0">
        {error && <p className="text-sm text-red-400">{error}</p>}
        <div className="flex gap-2">
          <input
            ref={codeInputRef}
            value={code}
            onChange={e => setCode(e.target.value)}
            placeholder="Scan or type barcode…"
            className="flex-1 px-3 py-3 text-sm rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
          />
          <input
            type="number" min="1" value={quantity} onChange={e => setQuantity(e.target.value)}
            className="w-16 px-2 py-3 text-sm text-center rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            type="button"
            onClick={() => setCameraOpen(true)}
            title="Use camera"
            className="w-11 shrink-0 rounded-xl bg-[#1F2937] border border-[#2D3748] flex items-center justify-center text-gray-300"
          >
            <Camera size={16} />
          </button>
        </div>
        <button type="button" onClick={() => setAdvancedOpen(o => !o)} className="flex items-center gap-1 text-xs text-gray-500">
          {advancedOpen ? <ChevronUp size={12} /> : <ChevronDown size={12} />} Lot / expiry
        </button>
        {advancedOpen && (
          <div className="grid grid-cols-2 gap-2">
            <input
              value={lotNumber} onChange={e => setLotNumber(e.target.value)} placeholder="Lot number"
              className="px-3 py-2 text-sm rounded-lg bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none"
            />
            <input
              type="date" value={expiryDate} onChange={e => setExpiryDate(e.target.value)}
              className="px-3 py-2 text-sm rounded-lg bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none"
            />
          </div>
        )}
        <button
          type="submit" disabled={scanning || !code.trim()}
          className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold hover:bg-blue-700 disabled:opacity-50"
        >
          {scanning ? 'Saving…' : 'Add'}
        </button>
      </form>

      {cameraOpen && (
        <CameraScanner onDecode={handleCameraDecode} onClose={() => setCameraOpen(false)} />
      )}
    </div>
  )
}
