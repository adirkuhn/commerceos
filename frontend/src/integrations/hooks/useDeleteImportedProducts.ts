import i18next from 'i18next'
import { useState } from 'react'
import * as api from '../api'
import type { Connection, DeleteAllImportedProductsResult } from '../types'
import { notifyProductsChanged } from '../../pms/productsBus'

export const DELETE_CONFIRMATION_WORD = 'DELETE'

// State for the "Delete imported products" modal of one connection. The
// modal owns it, so it resets whenever the modal closes (including Back).
export function useDeleteImportedProducts(token: string, connection: Connection | null) {
  const [confirmText, setConfirmText] = useState('')
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState('')
  const [result, setResult] = useState<DeleteAllImportedProductsResult | null>(null)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token || !connection) return
    setError('')
    setDeleting(true)
    try {
      setResult(await api.deleteAllImportedProducts(token, connection.id))
      notifyProductsChanged()
    } catch (err) {
      setError(err instanceof Error ? err.message : i18next.t('integrations:deleteImported.failed'))
    } finally {
      setDeleting(false)
    }
  }

  return {
    confirmText, setConfirmText, deleting, error, result, submit,
    confirmed: confirmText === DELETE_CONFIRMATION_WORD,
  }
}
