import { useMemo, useState } from 'react'
import { ClipboardCopy, Download, Mail, Printer } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { useAuth } from '../../auth/AuthContext'
import { useDateFormat } from '../../format/DateFormatProvider'
import { apiErrorText } from '../../api/apiErrorText'
import { downloadBlob } from '../../newsletter/download'
import type { EmailSupplierIssuesResult, SupplierIssue, SupplierIssueStatus } from '../types'
import { useSupplierIssues } from './useSupplierIssues'
import { SupplierIssueRow } from './SupplierIssueRow'
import { EmailSupplierDialog } from './EmailSupplierDialog'
import { SupplierIssuesPrintView } from './SupplierIssuesPrintView'
import { buildSupplierMessage, defaultSelection, issueKey, purchaseOrderRef, supplierIssuesCsv } from './supplierIssueText'

interface SupplierIssuesSectionProps {
  purchaseOrderId: string
  // When the PO was created -- quoted to the supplier.
  orderDate: string
  // False until a first delivery was booked: until then nothing can be
  // missing yet.
  hasCompletedBooking: boolean
}

const ACTION = 'flex items-center gap-1 px-2.5 py-1 rounded-lg border border-line bg-raised text-2xs text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors'

// What to claim from the supplier: missing, damaged and lost items of
// every finished delivery, with a claim status, and ways to send the list.
export function SupplierIssuesSection({ purchaseOrderId, orderDate, hasCompletedBooking }: SupplierIssuesSectionProps) {
  const { token } = useAuth()
  const { t } = useTranslation(['procurement', 'common'])
  const { formatDate } = useDateFormat()
  const { list, loading, error, updateStatus, email } = useSupplierIssues(token ?? '', purchaseOrderId)
  const [showResolved, setShowResolved] = useState(false)
  // null = the default (everything not resolved), until the user picks.
  const [picked, setPicked] = useState<Set<string> | null>(null)
  const [actionError, setActionError] = useState('')
  const [notice, setNotice] = useState('')
  const [emailOpen, setEmailOpen] = useState(false)
  const [printOpen, setPrintOpen] = useState(false)

  const issues = list?.issues ?? []
  const selected = picked ?? defaultSelection(issues)
  const resolvedCount = issues.filter(i => i.status === 'resolved').length
  const visible = showResolved ? issues : issues.filter(i => i.status !== 'resolved')
  const chosen = issues.filter(i => selected.has(issueKey(i)))
  const poRef = purchaseOrderRef(purchaseOrderId)
  const supplierName = list?.supplier?.name ?? null
  const message = useMemo(
    () => buildSupplierMessage(chosen, { supplierName, poRef, orderDate: formatDate(orderDate) }, t as TFunction<'procurement'>),
    [chosen, supplierName, poRef, orderDate, formatDate, t],
  )

  function toggle(issue: SupplierIssue) {
    const next = new Set(selected)
    if (next.has(issueKey(issue))) next.delete(issueKey(issue))
    else next.add(issueKey(issue))
    setPicked(next)
  }

  function toggleAll() {
    setPicked(visible.every(i => selected.has(issueKey(i))) ? new Set() : new Set(visible.map(issueKey)))
  }

  async function handleUpdate(issue: SupplierIssue, status: SupplierIssueStatus, note: string) {
    setActionError('')
    try {
      await updateStatus(issue, status, note)
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('supplierIssues.updateFailed'))
      throw err
    }
  }

  async function handleCopy() {
    setActionError('')
    setNotice('')
    try {
      await navigator.clipboard.writeText(message.body)
      setNotice(t('supplierIssues.actions.copied'))
    } catch {
      setActionError(t('supplierIssues.actions.copyFailed'))
    }
  }

  function handleCsv() {
    const csv = supplierIssuesCsv(chosen, t as TFunction<'procurement'>, iso => formatDate(iso))
    downloadBlob(new Blob([csv], { type: 'text/csv' }), `supplier-issues-${poRef.slice(1)}.csv`)
  }

  function handleSent(result: EmailSupplierIssuesResult) {
    setEmailOpen(false)
    setPicked(null)
    setNotice(t('supplierIssues.email.sent', { to: result.sent_to, count: result.claimed }))
  }

  const nothingChosen = chosen.length === 0

  return (
    <div className="rounded-xl border border-line overflow-hidden">
      <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-3 dark:bg-night-900/50 bg-gray-50 border-b border-line">
        <div className="min-w-0">
          <h3 className="text-xs font-semibold text-ink">{t('supplierIssues.title')}</h3>
          <p className="text-2xs text-ink-4">{t('supplierIssues.subtitle')}</p>
        </div>
        {issues.length > 0 && (
          <div className="flex flex-wrap items-center gap-1.5">
            <button type="button" onClick={handleCopy} disabled={nothingChosen} className={ACTION}>
              <ClipboardCopy size={11} /> {t('supplierIssues.actions.copy')}
            </button>
            <button type="button" onClick={handleCsv} disabled={nothingChosen} className={ACTION}>
              <Download size={11} /> {t('supplierIssues.actions.csv')}
            </button>
            <button type="button" onClick={() => setPrintOpen(true)} disabled={nothingChosen} className={ACTION}>
              <Printer size={11} /> {t('supplierIssues.actions.print')}
            </button>
            <button type="button" onClick={() => { setNotice(''); setEmailOpen(true) }} disabled={nothingChosen} className={ACTION}>
              <Mail size={11} /> {t('supplierIssues.actions.email')}
            </button>
          </div>
        )}
      </div>

      {list?.booking_in_progress && (
        <p className="text-2xs text-warning bg-amber-50 dark:bg-amber-500/10 px-4 py-1.5">{t('supplierIssues.bookingInProgress')}</p>
      )}
      {(error || actionError) && (
        <p role="alert" className="text-2xs text-danger bg-red-50 dark:bg-red-500/10 px-4 py-1.5">{error || actionError}</p>
      )}
      {notice && <p role="status" className="text-2xs text-success px-4 py-1.5">{notice}</p>}

      {loading && !list && <p className="px-4 py-6 text-center text-xs text-ink-4">{t('common:status.loading')}</p>}
      {list && issues.length === 0 && (
        <p className="px-4 py-6 text-center text-xs text-ink-4">
          {hasCompletedBooking ? t('supplierIssues.empty') : t('supplierIssues.noBookingYet')}
        </p>
      )}

      {issues.length > 0 && (
        <>
          <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-2 border-b border-line/60 text-2xs text-ink-3">
            <label className="flex items-center gap-1.5">
              <input type="checkbox" checked={visible.length > 0 && visible.every(i => selected.has(issueKey(i)))} onChange={toggleAll} />
              {t('supplierIssues.selectAll')}
            </label>
            <span>{t('supplierIssues.selectedCount', { count: chosen.length })}</span>
            {resolvedCount > 0 && (
              <label className="flex items-center gap-1.5">
                <input type="checkbox" checked={showResolved} onChange={e => setShowResolved(e.target.checked)} />
                {t('supplierIssues.showResolved', { count: resolvedCount })}
              </label>
            )}
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="sr-only">
                <tr>
                  <th />
                  <th>{t('supplierIssues.col.kind')}</th>
                  <th>{t('supplierIssues.col.item')}</th>
                  <th>{t('supplierIssues.col.qty')}</th>
                  <th>{t('supplierIssues.col.status')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line/50">
                {visible.map(issue => (
                  <SupplierIssueRow
                    key={issueKey(issue)}
                    issue={issue}
                    selected={selected.has(issueKey(issue))}
                    onToggle={() => toggle(issue)}
                    onUpdate={(status, note) => handleUpdate(issue, status, note)}
                  />
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {emailOpen && (
        <EmailSupplierDialog
          initialTo={list?.supplier?.email ?? ''}
          initialSubject={message.subject}
          initialBody={message.body}
          issues={chosen.map(i => ({ source: i.source, source_id: i.source_id }))}
          onSend={email}
          onSent={handleSent}
          onClose={() => setEmailOpen(false)}
        />
      )}
      {printOpen && (
        <SupplierIssuesPrintView issues={chosen} poRef={poRef} supplierName={supplierName} orderDate={orderDate} onClose={() => setPrintOpen(false)} />
      )}
    </div>
  )
}
