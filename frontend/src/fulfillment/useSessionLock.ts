import { useEffect, useRef } from 'react'

// Mirrors backend domain/oms SessionLockTTL -- a lock without a heartbeat
// for this long is free.
export const LOCK_TTL_MS = 2 * 60_000
// Two beats per TTL, so one missed request doesn't lose the lock.
export const HEARTBEAT_MS = 60_000

interface SessionLockOptions {
  sessionId: string | null
  me: string
  lockedBy?: string | null
  lockedAt?: string | null
  // False once the session is finished -- nothing to keep or release.
  active: boolean
  touch: (sessionId: string) => Promise<unknown>
  // Must survive the page closing (fetch keepalive).
  release: (sessionId: string) => Promise<unknown>
  // Someone took the lock over (the heartbeat came back 409).
  onLost: () => void
}

interface SessionLockState {
  holding: boolean
  // Who else actively holds the lock -- the screen shows the session
  // read-only while set. Null when free, stale, or ours.
  heldByOther: string | null
}

// Picking/packing session lock, client side. Opening a session only views
// it; the backend gives the lock to the first action. While we hold it we
// heartbeat it, and we release it the moment we leave the order (another
// order, Back, unmount, closing the tab) so browsing never keeps orders
// locked.
export function useSessionLock({ sessionId, me, lockedBy, lockedAt, active, touch, release, onLost }: SessionLockOptions): SessionLockState {
  const holding = active && !!sessionId && !!lockedBy && lockedBy === me
  const fresh = !!lockedAt && Date.now() - new Date(lockedAt).getTime() < LOCK_TTL_MS
  const heldByOther = active && lockedBy && lockedBy !== me && fresh ? lockedBy : null

  const callbacks = useRef({ touch, release, onLost })
  callbacks.current = { touch, release, onLost }

  useEffect(() => {
    if (!holding || !sessionId) return
    const interval = setInterval(() => {
      callbacks.current.touch(sessionId).catch(err => {
        if (err instanceof Error && /locked/i.test(err.message)) callbacks.current.onLost()
      })
    }, HEARTBEAT_MS)
    return () => clearInterval(interval)
  }, [holding, sessionId])

  useEffect(() => {
    if (!holding || !sessionId) return
    const releaseNow = () => {
      callbacks.current.release(sessionId).catch(err => {
        console.warn('useSessionLock: releasing the lock failed; it will expire on its own', err)
      })
    }
    window.addEventListener('pagehide', releaseNow)
    return () => {
      window.removeEventListener('pagehide', releaseNow)
      releaseNow()
    }
  }, [holding, sessionId])

  return { holding, heldByOther }
}
