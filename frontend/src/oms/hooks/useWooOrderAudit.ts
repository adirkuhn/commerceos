import i18next from 'i18next'
import { useState, useCallback } from 'react'
import { auditWooOrderDifferences } from '../orderDifferences'

// useWooOrderAudit compares every already-imported order of one Woo
// connection with Woo, one server batch at a time (cursor), so progress
// shows as it goes. onBatch runs after each batch (e.g. reload the list).
export function useWooOrderAudit(token: string, onBatch: () => void) {
  const [running, setRunning] = useState(false)
  const [done, setDone] = useState(false)
  const [checked, setChecked] = useState(0)
  const [failures, setFailures] = useState<string[]>([])
  const [error, setError] = useState<string | null>(null)

  const run = useCallback(async (connectionId: string) => {
    setRunning(true)
    setDone(false)
    setChecked(0)
    setFailures([])
    setError(null)
    try {
      let cursor = ''
      do {
        const result = await auditWooOrderDifferences(token, connectionId, cursor)
        setChecked(n => n + result.checked)
        setFailures(f => [...f, ...result.errors])
        onBatch()
        cursor = result.next_cursor
      } while (cursor)
      setDone(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('stockIssues:differences.auditFailed'))
    } finally {
      setRunning(false)
    }
  }, [token, onBatch])

  return { run, running, done, checked, failures, error }
}
