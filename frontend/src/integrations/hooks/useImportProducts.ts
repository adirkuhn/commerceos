import i18next from 'i18next'
import { useState } from 'react'
import * as api from '../api'
import type { Connection, ImportJob } from '../types'
import { useInitOnce } from '../../routing/useInitOnce'
import { normalizeCurrency } from '../integrationsPageHelpers'
import { notifyNotificationsChanged } from '../../notifications/notificationsBus'

// State for the "Import products" modal of one WooCommerce connection.
export function useImportProducts(token: string, connection: Connection | null, tenantDefaultCurrency: string | undefined) {
  const [warehouseId, setWarehouseId] = useState('')
  const [currency, setCurrency] = useState('')
  const [importing, setImporting] = useState(false)
  const [error, setError] = useState('')
  const [job, setJob] = useState<ImportJob | null>(null)

  // Default the import currency once per opened import modal (also after a
  // refresh on its URL, once the connection has loaded).
  useInitOnce(connection ? `import:${connection.id}` : null, !!connection, () => {
    setWarehouseId('')
    setError('')
    setJob(null)
    setCurrency(normalizeCurrency(connection?.default_currency) || normalizeCurrency(tenantDefaultCurrency))
  })

  // Enqueues a background job and returns immediately (202) instead of
  // blocking on the full sync, which for a real catalog can take minutes —
  // the job runs in cmd/scheduler; progress/completion shows up in the
  // notification bell, not here. The backend rejects a second import for
  // the same connection with 409 while one's already active, so this is
  // safe even if the button is clicked again in a new tab/session.
  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token || !connection) return
    setError('')
    setImporting(true)
    try {
      const started = await api.importWooProducts(token, connection.id, warehouseId, currency)
      setJob(started)
      notifyNotificationsChanged()
    } catch (err) {
      setError(err instanceof Error ? err.message : i18next.t('integrations:import.failed'))
    } finally {
      setImporting(false)
    }
  }

  return { warehouseId, setWarehouseId, currency, setCurrency, importing, error, job, submit }
}
