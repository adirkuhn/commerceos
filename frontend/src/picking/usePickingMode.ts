import { useEffect, useState } from 'react'
import * as settingsApi from '../settings/api'
import type { PickingMode } from '../settings/types'

interface UsePickingMode {
  // null until the tenant's settings load, so callers don't flash the
  // wrong mode's UI first.
  mode: PickingMode | null
  error: string
}

// The tenant's picking mode (Settings > Fulfillment). If settings can't be
// loaded, picking falls back to the default (scan) and reports the error.
export function usePickingMode(token: string): UsePickingMode {
  const [mode, setMode] = useState<PickingMode | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!token) return
    let cancelled = false
    settingsApi.getTenantSettings(token)
      .then(s => { if (!cancelled) setMode(s.picking_mode ?? 'scan') })
      .catch(err => {
        if (cancelled) return
        setError(err instanceof Error ? err.message : 'Failed to load picking mode')
        setMode('scan')
      })
    return () => { cancelled = true }
  }, [token])

  return { mode, error }
}
