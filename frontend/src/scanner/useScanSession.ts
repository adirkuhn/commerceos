import { useEffect, useRef, useState } from 'react'
import { useSessionLock } from '../fulfillment/useSessionLock'
import { readClaims } from '../auth/jwtClaims'
import { useMemberNames } from '../team/hooks/useMemberNames'
import { isAuthError } from '../auth/isAuthError'
import { useWakeLock } from './useWakeLock'

interface ScanSessionDetail {
  session: { status: string; locked_by?: string | null; locked_at?: string | null }
}

// The session-specific endpoints and wording the picking and packing
// screens plug into the shared session lifecycle below.
export interface ScanSessionApi<D> {
  get: (token: string, sessionId: string) => Promise<D>
  touch: (token: string, sessionId: string) => Promise<unknown>
  release: (token: string, sessionId: string) => Promise<unknown>
  takeOver: (token: string, sessionId: string) => Promise<unknown>
  deleteScan: (token: string, sessionId: string, scanId: string) => Promise<unknown>
  reset: (token: string, sessionId: string) => Promise<unknown>
  messages: { loadFailed: string; lockLost: string; resetFailed: string }
}

// A picking or packing session as the scanner works it: loading the
// detail, the shared error/busy state, staged actions that reload it, the
// lock, and start-over.
export function useScanSession<D extends ScanSessionDetail>(
  token: string, sessionId: string, onAuthError: () => void, sessionApi: ScanSessionApi<D>,
) {
  const [detail, setDetail] = useState<D | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [scanning, setScanning] = useState(false)
  const [resetConfirmOpen, setResetConfirmOpen] = useState(false)
  const [resetting, setResetting] = useState(false)
  const codeInputRef = useRef<HTMLInputElement>(null)

  const load = async () => {
    try {
      setDetail(await sessionApi.get(token, sessionId))
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : sessionApi.messages.loadFailed)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [sessionId])
  useEffect(() => { codeInputRef.current?.focus() }, [detail])

  // Opening only views; the first action takes the lock. While we hold it
  // we heartbeat it, and leaving the order releases it.
  const me = readClaims(token)
  const nameFor = useMemberNames(token)
  const { heldByOther } = useSessionLock({
    sessionId, me: me?.userId ?? '',
    lockedBy: detail?.session.locked_by, lockedAt: detail?.session.locked_at,
    active: !!detail && detail.session.status !== 'completed',
    touch: id => sessionApi.touch(token, id),
    release: id => sessionApi.release(token, id),
    onLost: () => setError(sessionApi.messages.lockLost),
  })
  const readOnly = heldByOther !== null

  async function takeOver() {
    try {
      await sessionApi.takeOver(token, sessionId)
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to take over')
    }
  }
  useWakeLock()

  // Runs one staged action (scan, substitute, refund, extra item) and
  // reloads the session.
  async function run(action: () => Promise<unknown>, failure: string): Promise<boolean> {
    setError('')
    setScanning(true)
    try {
      await action()
      await load()
      return true
    } catch (err) {
      if (isAuthError(err)) {
        onAuthError()
        return false
      }
      setError(err instanceof Error ? err.message : failure)
      return false
    } finally {
      setScanning(false)
      codeInputRef.current?.focus()
    }
  }

  async function deleteScan(scanId: string) {
    try {
      await sessionApi.deleteScan(token, sessionId, scanId)
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to delete scan')
    }
  }

  // onReset closes whatever per-line sheets the screen has open.
  async function resetSession(onReset: () => void) {
    setResetting(true)
    try {
      await sessionApi.reset(token, sessionId)
      setResetConfirmOpen(false)
      onReset()
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : sessionApi.messages.resetFailed)
    } finally {
      setResetting(false)
    }
  }

  return {
    detail, loading, error, setError, scanning, setScanning, codeInputRef, load, run, deleteScan,
    me, nameFor, heldByOther, readOnly, takeOver,
    resetConfirmOpen, setResetConfirmOpen, resetting, resetSession,
  }
}
