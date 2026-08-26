import { useEffect } from 'react'

// Keeps the screen from sleeping while a picking session is open --
// scanning is hands-on-device only between scans, so the OS's normal
// screen-off timeout would otherwise lock a picker out mid-session.
// Feature-detected: iOS Safari < 16.4 and older Android WebViews have no
// Wake Lock API, so this silently no-ops there rather than breaking the
// screen.
export function useWakeLock(): void {
  useEffect(() => {
    if (!navigator.wakeLock) return

    let cancelled = false
    let sentinel: WakeLockSentinel | undefined

    async function acquire() {
      try {
        sentinel = await navigator.wakeLock.request('screen')
      } catch {
        // Rejected (tab backgrounded, battery saver, etc) -- reacquired on
        // the next visibilitychange, nothing to surface to the user.
      }
    }

    function onVisibilityChange() {
      // The lock is released automatically whenever the tab backgrounds
      // (switching to the camera scanner, another app, screen truly off),
      // so it must be re-requested each time the picking tab comes back.
      if (document.visibilityState === 'visible' && !cancelled) acquire()
    }

    acquire()
    document.addEventListener('visibilitychange', onVisibilityChange)

    return () => {
      cancelled = true
      document.removeEventListener('visibilitychange', onVisibilityChange)
      sentinel?.release()
    }
  }, [])
}
