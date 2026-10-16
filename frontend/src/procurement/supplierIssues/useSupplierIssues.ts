import i18next from 'i18next'
import { useCallback, useEffect, useState } from 'react'
import * as api from '../api'
import type { EmailSupplierIssuesInput, EmailSupplierIssuesResult, SupplierIssueList, SupplierIssueRef, SupplierIssueStatus } from '../types'
import { apiErrorText } from '../../api/apiErrorText'
import { issueKey } from './supplierIssueText'

// A purchase order's supplier issues, derived server-side on every load.
// updateStatus and email throw on failure so the caller can show it where
// the user acted.
export function useSupplierIssues(token: string, purchaseOrderId: string) {
  const [list, setList] = useState<SupplierIssueList | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      setList(await api.listSupplierIssues(token, purchaseOrderId))
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('procurement:supplierIssues.loadFailed'))
    } finally {
      setLoading(false)
    }
  }, [token, purchaseOrderId])

  useEffect(() => { load() }, [load])

  const updateStatus = useCallback(async (ref: SupplierIssueRef, status: SupplierIssueStatus, note: string) => {
    const updated = await api.updateSupplierIssueStatus(token, purchaseOrderId, ref, status, note)
    setList(prev => prev && {
      ...prev,
      issues: prev.issues.map(i => (issueKey(i) === issueKey(updated) ? updated : i)),
    })
  }, [token, purchaseOrderId])

  const email = useCallback(async (input: EmailSupplierIssuesInput): Promise<EmailSupplierIssuesResult> => {
    const result = await api.emailSupplierIssues(token, purchaseOrderId, input)
    await load()
    return result
  }, [token, purchaseOrderId, load])

  return { list, loading, error, refresh: load, updateStatus, email }
}
