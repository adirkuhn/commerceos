import { useState, useEffect, useRef } from 'react'
import * as packingApi from '../scanner/api'
import type { PackingSessionDetail, PickingSessionLineItem } from '../scanner/types'
import type { SubstituteRowInput } from './LineChangePanel'
import type { AddItemInput } from './AddItemPanel'
import { playErrorSound } from '../scanner/errorSound'

interface PackActionsOptions {
  token: string | null
  session: PackingSessionDetail | null
  loadSession: (sessionId: string) => Promise<void>
  setError: (message: string) => void
}

export interface PackLine {
  lineItem: PickingSessionLineItem
  packedSoFar: number
}

// Everything staff do to an open packing session before finishing it: scan
// into the input, or open a line (pack/substitute/refund) or the "add an
// item" side panel; plus removing a staged scan.
export function usePackActions({ token, session, loadSession, setError }: PackActionsOptions) {
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [scanErrorAlert, setScanErrorAlert] = useState<string | null>(null)
  const codeRef = useRef<HTMLInputElement>(null)

  // The side panel: one line (pack/substitute/refund), or "add an item".
  const [packLine, setPackLine] = useState<PackLine | null>(null)
  const [addItemOpen, setAddItemOpen] = useState(false)

  useEffect(() => { codeRef.current?.focus() }, [session, packLine])

  // Same "instant increment on scan, blocking modal on any mis-scan"
  // pattern as Picking.tsx's handleScanSubmit -- see the fix note there for
  // why a mis-scan must never fall through to a non-blocking inline error.
  async function submitScan() {
    const scannedCode = code.trim()
    if (!scannedCode || !session || !token || scanErrorAlert) return
    setError('')
    setBusy(true)
    try {
      await packingApi.scanPackItem(token, session.session.id, { code: scannedCode, quantity: 1 })
      setCode('')
      await loadSession(session.session.id)
    } catch (err) {
      playErrorSound()
      setScanErrorAlert(err instanceof Error ? err.message : 'Item not on this order')
    } finally {
      setBusy(false)
      codeRef.current?.focus()
    }
  }

  function dismissScanError() {
    setScanErrorAlert(null)
    codeRef.current?.focus()
  }

  function openLine(li: PickingSessionLineItem, packedSoFar: number) {
    if (!li.sku_id) return
    setAddItemOpen(false)
    setPackLine({ lineItem: li, packedSoFar })
  }

  function openAddItem() {
    setPackLine(null)
    setAddItemOpen(true)
  }

  function closePanels() {
    setPackLine(null)
    setAddItemOpen(false)
  }

  // Runs one staged change against the session and reloads it -- every
  // panel action (pack, substitute, refund, add item) goes through here.
  async function stageChange(action: (sessionId: string, token: string) => Promise<unknown>, failure: string) {
    if (!session || !token) return
    setBusy(true)
    setError('')
    try {
      await action(session.session.id, token)
      setPackLine(null)
      setAddItemOpen(false)
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : failure)
    } finally {
      setBusy(false)
      codeRef.current?.focus()
    }
  }

  function confirmPack(quantity: number) {
    const skuId = packLine?.lineItem.sku_id
    if (!skuId) return
    stageChange((id, t) => packingApi.scanPackItem(t, id, { sku_id: skuId, quantity }), 'Failed to record pack')
  }

  function confirmSubstitute(rows: SubstituteRowInput[]) {
    const lineItemId = packLine?.lineItem.id
    if (!lineItemId) return
    stageChange(async (id, t) => {
      const lines = await Promise.all(rows.map(async r => ({
        sku_id: (await packingApi.resolvePackScanCode(t, id, r.code)).sku_id,
        substitute_quantity: r.substituteQuantity, original_quantity_covered: r.coveredQuantity, return_to_stock: r.returnToStock,
      })))
      return packingApi.substitutePackLine(t, id, { line_item_id: lineItemId, lines })
    }, 'Failed to record substitute')
  }

  function confirmRefund(quantity: number, reason: string) {
    const lineItemId = packLine?.lineItem.id
    if (!lineItemId) return
    stageChange((id, t) => packingApi.refundPackLine(t, id, { line_item_id: lineItemId, quantity, reason: reason || undefined }), 'Failed to record refund')
  }

  function confirmAddItem(item: AddItemInput) {
    stageChange((id, t) => packingApi.scanPackItem(t, id, {
      sku_id: item.skuId, quantity: item.quantity, action_type: item.freebie ? 'freebie' : 'new_item',
      price_minor_units: item.priceMinorUnits, reason: item.name,
    }), 'Failed to add item')
  }

  async function deleteScan(scanId: string) {
    if (!session || !token) return
    setError('')
    try {
      await packingApi.deletePackScan(token, session.session.id, scanId)
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove scan')
    }
  }

  return {
    code, setCode, busy, codeRef, submitScan, scanErrorAlert, dismissScanError,
    packLine, openLine, closeLine: () => setPackLine(null),
    addItemOpen, openAddItem, closeAddItem: () => setAddItemOpen(false), closePanels,
    confirmPack, confirmSubstitute, confirmRefund, confirmAddItem, deleteScan,
  }
}
