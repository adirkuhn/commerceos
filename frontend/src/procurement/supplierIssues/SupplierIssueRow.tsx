import { useState } from 'react'
import { Pencil } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import type { SupplierIssue, SupplierIssueKind, SupplierIssueStatus } from '../types'
import { useDateFormat } from '../../format/DateFormatProvider'
import { issueItemName } from './supplierIssueText'

export const SUPPLIER_ISSUE_STATUSES: SupplierIssueStatus[] = ['open', 'claimed', 'resolved']

const KIND_STYLE: Record<SupplierIssueKind, string> = {
  missing: 'bg-amber-100 dark:bg-warning/10 text-warning',
  damaged: 'bg-red-50 dark:bg-danger/10 text-danger',
  lost: 'bg-fill text-ink-3',
}

interface SupplierIssueRowProps {
  issue: SupplierIssue
  selected: boolean
  onToggle: () => void
  // Rejects on failure; the section shows the error.
  onUpdate: (status: SupplierIssueStatus, note: string) => Promise<void>
}

export function SupplierIssueRow({ issue, selected, onToggle, onUpdate }: SupplierIssueRowProps) {
  const { t } = useTranslation(['procurement', 'common'])
  const { formatDate } = useDateFormat()
  const [editingNote, setEditingNote] = useState(false)
  const [note, setNote] = useState(issue.status_note)
  const name = issueItemName(issue)
  const ourName = issue.product_name && issue.product_name !== name ? issue.product_name : null

  async function saveNote() {
    try {
      await onUpdate(issue.status, note.trim())
      setEditingNote(false)
    } catch {
      // the section shows the error; keep the input open to retry
    }
  }

  return (
    <tr aria-label={name} className={clsx('align-top', issue.status === 'resolved' && 'opacity-60')}>
      <td className="px-3 py-2">
        <input type="checkbox" checked={selected} onChange={onToggle} aria-label={t('supplierIssues.selectIssue', { item: name })} />
      </td>
      <td className="px-2 py-2">
        <span className={clsx('px-2 py-0.5 rounded-sm text-2xs font-semibold whitespace-nowrap', KIND_STYLE[issue.kind])}>
          {t(`supplierIssues.kind.${issue.kind}`)}
        </span>
      </td>
      <td className="px-2 py-2 min-w-0">
        <p className="text-xs text-ink break-words">{name}</p>
        <p className="text-2xs text-ink-4 break-words">
          {[issue.supplier_sku, ourName, issue.sku_code].filter(Boolean).join(' · ')}
        </p>
        {issue.kind === 'missing' && issue.expected_qty != null && (
          <p className="text-2xs text-ink-4">{t('supplierIssues.orderedReceived', { expected: issue.expected_qty, received: issue.received_qty ?? 0 })}</p>
        )}
        {issue.reason && <p className="text-2xs text-ink-3 break-words">{issue.reason}</p>}
        <p className="text-2xs text-ink-4">
          {formatDate(issue.occurred_at)}{issue.reported_by && ` · ${t('supplierIssues.reportedBy', { name: issue.reported_by })}`}
        </p>
      </td>
      <td className="px-2 py-2 text-xs text-ink tabular-nums text-right">{issue.quantity}</td>
      <td className="px-3 py-2 space-y-1">
        <select
          value={issue.status}
          onChange={e => { onUpdate(e.target.value as SupplierIssueStatus, issue.status_note).catch(() => {}) }}
          aria-label={t('supplierIssues.statusLabel', { item: name })}
          className="w-full rounded-lg border border-line bg-raised px-2 py-1 text-2xs text-ink"
        >
          {SUPPLIER_ISSUE_STATUSES.map(s => <option key={s} value={s}>{t(`supplierIssues.status.${s}`)}</option>)}
        </select>
        {editingNote ? (
          <input
            autoFocus value={note} maxLength={1000}
            onChange={e => setNote(e.target.value)}
            onKeyDown={e => {
              if (e.key === 'Enter') { e.preventDefault(); saveNote() }
              if (e.key === 'Escape') { setNote(issue.status_note); setEditingNote(false) }
            }}
            onBlur={() => { if (note.trim() !== issue.status_note) saveNote(); else setEditingNote(false) }}
            aria-label={t('supplierIssues.note.label', { item: name })}
            placeholder={t('supplierIssues.note.placeholder')}
            className="w-full rounded-lg border border-line bg-raised px-2 py-1 text-2xs text-ink"
          />
        ) : (
          <button
            type="button" onClick={() => { setNote(issue.status_note); setEditingNote(true) }}
            aria-label={issue.status_note ? t('supplierIssues.note.edit') : t('supplierIssues.note.add')}
            className="flex items-center gap-1 text-2xs text-ink-4 hover:text-ink-2 text-left break-words"
          >
            <Pencil size={10} className="shrink-0" /> {issue.status_note || t('supplierIssues.note.add')}
          </button>
        )}
      </td>
    </tr>
  )
}
