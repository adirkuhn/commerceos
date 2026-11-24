import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Trash2, FileText, AlertTriangle, RefreshCw } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { pollParsingJob } from './pollParsingJob'
import * as procurementApi from './api'
import type { PurchaseOrderDocument } from './types'
import { useConfirm } from '../components/common/useConfirm'
import { friendlyParsingError } from './procurementHelpers'
import { apiErrorText } from '../api/apiErrorText'

// ── Documents ────────────────────────────────────────────────────────────────

interface DocumentsSectionProps {
  documents: PurchaseOrderDocument[]
  onDelete: (id: string) => Promise<void>
  onRefresh: () => void
}

export function DocumentsSection({ documents, onDelete, onRefresh }: DocumentsSectionProps) {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { t } = useTranslation(['procurement', 'common'])
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [reparsingId, setReparsingId] = useState<string | null>(null)
  const [reparseError, setReparseError] = useState<{ id: string; message: string } | null>(null)

  async function handleDelete(doc: PurchaseOrderDocument) {
    if (!(await confirm({ title: t('docs.deleteTitle', { name: doc.file_name }), message: t('docs.deleteMessage'), confirmLabel: t('docs.deleteConfirm'), destructive: true }))) return
    setDeletingId(doc.id)
    try {
      await onDelete(doc.id)
    } finally {
      setDeletingId(null)
    }
  }

  async function handleReparse(doc: PurchaseOrderDocument) {
    setReparseError(null)
    setReparsingId(doc.id)
    try {
      const job = await procurementApi.retryDocumentParsingJob(token ?? '', doc.id)
      const finalJob = await pollParsingJob(() => procurementApi.getDocumentParsingJob(token ?? '', job.id))
      if (finalJob.status === 'failed') {
        setReparseError({ id: doc.id, message: friendlyParsingError(finalJob.error_message) })
      }
      onRefresh()
    } catch (err) {
      setReparseError({ id: doc.id, message: apiErrorText(err) ?? t('docs.reparseFailed') })
    } finally {
      setReparsingId(null)
    }
  }

  return (
    <div className="rounded-xl border border-line overflow-hidden">
      <div className="px-4 py-3 bg-sunken border-b border-line">
        <h3 className="text-xs font-semibold text-ink">{t('docs.title')}</h3>
      </div>
      <div className="divide-y divide-line">
        {documents.map(doc => (
          <div key={doc.id} className="px-4 py-2.5">
            <div className="flex items-center gap-2 text-xs text-ink-2">
              <FileText size={13} className="text-ink-4 shrink-0" />
              <a href={doc.file_url} target="_blank" rel="noreferrer" className="flex-1 min-w-0 truncate hover:underline">
                {doc.file_name}
              </a>
              {doc.is_potential_duplicate && (
                <span className="flex items-center gap-1 px-1.5 py-0.5 rounded-sm text-2xs font-semibold text-warning bg-amber-500/10 shrink-0">
                  <AlertTriangle size={10} /> {t('docs.possibleDuplicate')}
                </span>
              )}
              <button aria-label={t('docs.reparse')}
                type="button"
                title={t('docs.reparse')}
                onClick={() => handleReparse(doc)}
                disabled={reparsingId === doc.id}
                className="p-1 rounded text-ink-4 hover:text-accent disabled:opacity-50 transition-colors shrink-0"
              >
                <RefreshCw size={13} className={reparsingId === doc.id ? 'animate-spin' : ''} />
              </button>
              <button aria-label={t('common:actions.delete')}
                type="button"
                title={t('common:actions.delete')}
                onClick={() => handleDelete(doc)}
                disabled={deletingId === doc.id}
                className="p-1 rounded text-ink-4 hover:text-danger disabled:opacity-50 transition-colors shrink-0"
              >
                <Trash2 size={13} />
              </button>
            </div>
            {reparseError?.id === doc.id && (
              <p className="mt-1 text-2xs text-danger pl-[21px]">{reparseError.message}</p>
            )}
          </div>
        ))}
      </div>
      {confirmDialog}
    </div>
  )
}
