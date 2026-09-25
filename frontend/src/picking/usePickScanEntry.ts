import { useEffect, useRef, useState } from 'react'
import * as pickingApi from '../scanner/api'
import type { PickingSessionDetail } from '../scanner/types'
import { pickedQuantityByLineItem } from '../scanner/pickingMissingItems'
import { playErrorSound } from '../scanner/errorSound'
import type { PickLineSheet } from './usePickLineSheet'

interface PickScanEntryOptions {
  token: string | null
  session: PickingSessionDetail | null
  loadSession: (sessionId: string) => Promise<void>
  setError: (message: string) => void
  sheet: Pick<PickLineSheet, 'pickLine' | 'pickQty' | 'close'>
}

// Recording picks on the open session: the scan box (with its blocking
// error alert), the pick sheet's confirm, and removing a staged scan.
export function usePickScanEntry({ token, session, loadSession, setError, sheet }: PickScanEntryOptions) {
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [scanErrorAlert, setScanErrorAlert] = useState<string | null>(null)
  const codeRef = useRef<HTMLInputElement>(null)
  const { pickLine, pickQty } = sheet

  useEffect(() => { codeRef.current?.focus() }, [session, pickLine])

  // A scanned/typed code that matches one of this order's own line items
  // marks one unit picked immediately -- no confirm sheet, just an
  // increment (see the picking screens' shared design doc for why: the
  // sheet is now reserved for a deliberate tap on the line itself, to pick
  // a specific quantity or substitute). Already-fully-picked (over-scan) or
  // resolved-but-not-on-this-order both block further scanning behind
  // scanErrorAlert until the picker explicitly confirms the error --
  // mis-scans must never silently fall through to the freebie/new-item
  // flow.
  async function handleScanSubmit() {
    const scannedCode = code.trim()
    if (!scannedCode || !session || !token || scanErrorAlert) return
    setError('')
    setBusy(true)
    try {
      const resolved = await pickingApi.resolvePickScanCode(token, session.session.id, scannedCode)
      setCode('')
      const li = session.line_items.find(l => l.sku_id === resolved.sku_id)
      if (li) {
        const pickedSoFar = pickedQuantityByLineItem(session.scans).get(li.id) ?? 0
        if (pickedSoFar >= li.quantity) {
          playErrorSound()
          setScanErrorAlert('Already fully picked')
          return
        }
      }
      // No line item already carries this sku_id -- still worth trying the
      // server: it may auto-link one of this order's still-unmatched Woo
      // lines (product name matches the scanned item) rather than this
      // always being a genuine "wrong order" scan (see
      // ScanPickItemUseCase.autoLinkUnmatchedLine).
      try {
        await pickingApi.scanPickItem(token, session.session.id, {
          sku_id: resolved.sku_id, action_type: 'pick', quantity: 1,
        })
        await loadSession(session.session.id)
      } catch {
        playErrorSound()
        setScanErrorAlert('Item not on this order')
      }
    } catch {
      // Code didn't resolve to any known SKU at all -- must block the same
      // way as the "item not on this order" / "already fully picked" cases
      // above, not just show inline text a picker can scan straight past
      // (see PickingScreen.tsx's resolveAndRoute, which already blocks here).
      playErrorSound()
      setScanErrorAlert('Barcode not recognized')
    } finally {
      setBusy(false)
      codeRef.current?.focus()
    }
  }

  function dismissScanError() {
    setScanErrorAlert(null)
    codeRef.current?.focus()
  }

  async function confirmPick() {
    if (!pickLine || !session || !token || !pickLine.lineItem.sku_id) return
    setBusy(true)
    setError('')
    try {
      await pickingApi.scanPickItem(token, session.session.id, {
        sku_id: pickLine.lineItem.sku_id, action_type: 'pick', quantity: pickQty,
      })
      sheet.close()
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to record pick')
    } finally {
      setBusy(false)
      codeRef.current?.focus()
    }
  }

  async function handleDeleteScan(scanId: string) {
    if (!session || !token) return
    setError('')
    try {
      await pickingApi.deletePickScan(token, session.session.id, scanId)
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove scan')
    }
  }

  return { code, setCode, busy, scanErrorAlert, codeRef, handleScanSubmit, dismissScanError, confirmPick, handleDeleteScan }
}
