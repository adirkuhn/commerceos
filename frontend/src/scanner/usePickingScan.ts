import { useState, type FormEvent, type RefObject } from 'react'
import * as api from './api'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { isAuthError } from '../auth/isAuthError'
import { pickedQuantityByLineItem } from './pickingMissingItems'
import { useScanInput } from './useScanInput'
import type { PickingSessionDetail } from './types'

interface PickingScanOptions {
  token: string
  sessionId: string
  detail: PickingSessionDetail | null
  codeInputRef: RefObject<HTMLInputElement>
  // Stages one pick and reloads the session (useScanSession's run).
  stage: (input: Parameters<typeof api.scanPickItem>[2]) => Promise<boolean>
  setError: (message: string) => void
  setScanning: (scanning: boolean) => void
  onAuthError: () => void
}

// Picking's two-step scan: resolve the code to a SKU, then route it to the
// order's line (or refuse it), with a name/code search fallback.
export function usePickingScan({ token, sessionId, detail, codeInputRef, stage, setError, setScanning, onAuthError }: PickingScanOptions) {
  const scanInput = useScanInput(codeInputRef)
  const { code, setCode, scanErrorAlert, showScanError } = scanInput
  const [candidates, setCandidates] = useState<SKUCandidate[]>([])

  // A scanned/typed code matching one of this order's own line items marks
  // one unit picked immediately -- no confirm sheet, just an increment (the
  // sheet is reserved for a deliberate tap on the line itself, to pick a
  // specific quantity or substitute -- see setPickLine's row-click callers
  // in PickingScreen). Already-fully-picked (over-scan) or resolved-but-not-
  // on-this-order both block further scanning behind scanErrorAlert until
  // the picker explicitly confirms the error -- mis-scans must never
  // silently fall through to the freebie/new-item flow. A resolve miss
  // still falls back to a partial name/code/barcode search (same as
  // Booking); only once that search also comes up empty does it become a
  // blocking error too.
  // weightGrams is set only when the scanned code was a weight-embedded
  // (butcher/deli scale) barcode -- see ResolvePickScanCodeUseCase. The
  // pick quantity becomes the scanned package's actual weight (in kg)
  // instead of a flat 1 unit; a per_weight SKU can only ever reach this
  // function via that path (ScanPickItemUseCase rejects any other
  // resolution for one), so weightGrams is never nil here for one.
  async function routeResolvedSku(skuId: string, weightGrams?: number | null) {
    const quantity = weightGrams != null ? weightGrams / 1000 : 1
    const li = detail?.line_items.find(l => l.sku_id === skuId)
    if (li) {
      const pickedSoFar = pickedQuantityByLineItem(detail?.scans ?? []).get(li.id) ?? 0
      if (pickedSoFar >= li.quantity) {
        showScanError('Already fully picked')
        return
      }
      await stage({ sku_id: skuId, action_type: 'pick', quantity })
      return
    }
    // No line item already carries this sku_id -- still worth trying the
    // server: it may auto-link one of this order's still-unmatched Woo
    // lines (product name matches the scanned item) rather than this
    // always being a genuine "wrong order" scan (see
    // ScanPickItemUseCase.autoLinkUnmatchedLine).
    const ok = await stage({ sku_id: skuId, action_type: 'pick', quantity })
    if (!ok) {
      setError('')
      showScanError('Item not on this order')
    }
  }

  async function resolveAndRoute(scannedCode: string) {
    if (scanErrorAlert) return
    setError('')
    setScanning(true)
    try {
      const resolved = await api.resolvePickScanCode(token, sessionId, scannedCode)
      setCode('')
      routeResolvedSku(resolved.sku_id, resolved.weight_grams)
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      try {
        const found = await searchSKUCandidates(token, scannedCode)
        if (found.length > 0) {
          setCode('')
          setCandidates(found)
          return
        }
      } catch {
        // fall through to the blocking error below
      }
      showScanError('Barcode not recognized')
    } finally {
      setScanning(false)
      codeInputRef.current?.focus()
    }
  }

  function selectCandidate(c: SKUCandidate) {
    // A name/code text match carries no real weight -- same rule
    // ScanPickItemUseCase enforces server-side for a barcode/SKU-code
    // scan, applied here too so this fuzzy-search fallback can't bypass it.
    if (c.sku.pricing_mode === 'per_weight') {
      setCandidates([])
      showScanError('Priced by weight — scan its scale barcode, not a name/code match')
      return
    }
    setCandidates([])
    setCode('')
    routeResolvedSku(c.sku.id)
  }

  function handleScan(e: FormEvent) {
    e.preventDefault()
    if (!code.trim()) return
    resolveAndRoute(code.trim())
  }

  function handleCameraDecode(text: string) {
    scanInput.setCameraOpen(false)
    resolveAndRoute(text)
  }

  return { ...scanInput, candidates, selectCandidate, handleScan, handleCameraDecode }
}
