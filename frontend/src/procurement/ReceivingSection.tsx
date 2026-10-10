import { useState, useEffect, useRef } from 'react'
import { X, Upload, PackageCheck, FileText, AlertTriangle, ClipboardList, Download } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useReceivingRecords } from './hooks/useReceivingRecords'
import { pollParsingJob } from './pollParsingJob'
import * as procurementApi from './api'
import type { PurchaseOrderLineItem, ReceivingRecordDetail, ReceivingDiscrepancy } from './types'
import type { Warehouse } from '../inventory/types'
import { Dialog } from '../components/common/Dialog'
import { DISCREPANCY_TYPES, RECEIVING_STATUS_STYLE, discrepancyLabel, receivingStatusLabel } from './procurementHelpers'
import { DiscrepancyRow, FlagDamagedItemForm } from './DiscrepancyRow'
import { useDateFormat } from '../format/DateFormatProvider'
import { apiErrorText } from '../api/apiErrorText'

// ── Receiving ────────────────────────────────────────────────────────────────

interface ReceivingSectionProps {
  purchaseOrderId: string
  // Nil until stock has been received against this PO at least once --
  // ReceiveDiscrepancyForm prompts for one inline the first time, via
  // onAssignWarehouse.
  warehouseId: string | null
  warehouses: Warehouse[]
  onAssignWarehouse: (warehouseId: string) => Promise<void>
  lineItems: PurchaseOrderLineItem[]
}

export function ReceivingSection({ purchaseOrderId, warehouseId, warehouses, onAssignWarehouse, lineItems }: ReceivingSectionProps) {
  const { token } = useAuth()
  const { records, loading, error, refresh, upload } = useReceivingRecords(token ?? '', purchaseOrderId)
  const { t } = useTranslation(['procurement', 'common'])
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [uploadStatus, setUploadStatus] = useState<'idle' | 'uploading' | 'parsing'>('idle')
  const [uploadError, setUploadError] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  async function handleFileSelected(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    setUploadError('')
    setUploadStatus('uploading')
    try {
      const job = await upload(file)
      setUploadStatus('parsing')
      const finalJob = await pollParsingJob(() => procurementApi.getReceivingParsingJob(token ?? '', job.id))
      if (finalJob.status === 'failed') {
        setUploadError(finalJob.error_message || t('receiving.parseFailed'))
      }
      refresh()
    } catch (err) {
      setUploadError(apiErrorText(err) ?? t('receiving.uploadFailed'))
    } finally {
      setUploadStatus('idle')
    }
  }

  return (
    <div className="rounded-xl border border-line overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 dark:bg-night-900/50 bg-gray-50 border-b border-line">
        <h3 className="text-xs font-semibold text-ink">{t('receiving.title')}</h3>
        <div className="flex items-center gap-2">
          {uploadStatus === 'uploading' && <span className="text-2xs text-ink-4">{t('receiving.uploading')}</span>}
          {uploadStatus === 'parsing' && <span className="text-2xs text-warning">{t('receiving.parsing')}</span>}
          <input ref={fileInputRef} type="file" aria-label={t('receiving.uploadInvoice')} accept=".pdf,.csv,.xlsx" className="hidden" onChange={handleFileSelected} />
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploadStatus !== 'idle'}
            className="flex items-center gap-1 px-2.5 py-1 rounded-lg border border-line bg-raised text-2xs text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
          >
            <Upload size={11} /> {t('receiving.uploadInvoice')}
          </button>
        </div>
      </div>
      {(uploadError || error) && (
        <p className="text-2xs text-danger bg-red-50 dark:bg-red-500/10 px-4 py-1.5">{uploadError || error}</p>
      )}
      <div className="divide-y divide-line/50">
        {loading && <p className="px-4 py-6 text-center text-xs text-ink-4">{t('common:status.loading')}</p>}
        {!loading && records.length === 0 && (
          <p className="px-4 py-6 text-center text-xs text-ink-4">{t('receiving.empty')}</p>
        )}
        {records.map(record => (
          <div key={record.id}>
            <button
              type="button"
              onClick={() => setExpandedId(expandedId === record.id ? null : record.id)}
              className="w-full flex items-center gap-3 px-4 py-2.5 text-left hover:bg-hover/40 transition-colors"
            >
              <FileText size={13} className="text-ink-4 shrink-0" />
              <span className="flex-1 text-xs text-ink truncate">{record.invoice_file_name}</span>
              {record.is_potential_duplicate && (
                <span
                  title={t('receiving.duplicateTitle')}
                  className="flex items-center gap-1 px-2 py-0.5 rounded-full text-2xs font-medium bg-amber-100 dark:bg-amber-500/10 text-warning"
                >
                  <AlertTriangle size={10} /> {t('receiving.possibleDuplicate')}
                </span>
              )}
              <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium capitalize', RECEIVING_STATUS_STYLE[record.status])}>
                {receivingStatusLabel(record.status)}
              </span>
            </button>
            {expandedId === record.id && (
              <div className="px-4 pb-3 dark:bg-night-950/40 bg-gray-50/60">
                <ReceivingRecordCard
                  recordId={record.id}
                  lineItems={lineItems}
                  warehouseId={warehouseId}
                  warehouses={warehouses}
                  onAssignWarehouse={onAssignWarehouse}
                  invoiceFileName={record.invoice_file_name}
                  onCompleted={() => { refresh() }}
                />
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

interface ReceivingRecordCardProps {
  recordId: string
  lineItems: PurchaseOrderLineItem[]
  warehouseId: string | null
  warehouses: Warehouse[]
  onAssignWarehouse: (warehouseId: string) => Promise<void>
  invoiceFileName: string
  onCompleted: () => void
}

function ReceivingRecordCard({ recordId, lineItems, warehouseId, warehouses, onAssignWarehouse, invoiceFileName, onCompleted }: ReceivingRecordCardProps) {
  const { token } = useAuth()
  const [detail, setDetail] = useState<ReceivingRecordDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [completing, setCompleting] = useState(false)
  const [damagedFormOpen, setDamagedFormOpen] = useState(false)
  const [reportOpen, setReportOpen] = useState(false)
  const { t } = useTranslation(['procurement', 'common'])

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      setDetail(await procurementApi.getReceivingRecord(token ?? '', recordId))
    } catch (err) {
      setError(apiErrorText(err) ?? t('receiving.loadFailed'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [recordId])

  async function handleComplete() {
    setCompleting(true)
    setError('')
    try {
      await procurementApi.completeReceiving(token ?? '', recordId)
      await load()
      onCompleted()
    } catch (err) {
      setError(apiErrorText(err) ?? t('receiving.completeFailed'))
    } finally {
      setCompleting(false)
    }
  }

  if (loading) return <p className="text-2xs text-ink-4 py-2">{t('common:status.loading')}</p>
  if (!detail) return error ? <p className="text-2xs text-danger py-2">{error}</p> : null

  const record = detail.receiving_record
  const isApplied = record.status === 'applied'

  return (
    <div className="space-y-2 pt-1">
      {detail.discrepancies.length === 0 ? (
        <p className="text-2xs text-ink-3">
          {t('receiving.noDiscrepancies')}
        </p>
      ) : (
        <div className="space-y-2">
          {detail.discrepancies.map(d => (
            <DiscrepancyRow
              key={d.id}
              discrepancy={d}
              allDiscrepancies={detail.discrepancies}
              lineItem={lineItems.find(li => li.id === d.line_item_id)}
              warehouseId={warehouseId}
              warehouses={warehouses}
              onAssignWarehouse={onAssignWarehouse}
              invoiceFileName={invoiceFileName}
              disabled={isApplied}
              onReload={load}
            />
          ))}
        </div>
      )}
      {error && <p className="text-2xs text-danger">{error}</p>}
      {!isApplied && (
        damagedFormOpen ? (
          <FlagDamagedItemForm recordId={recordId} lineItems={lineItems} onDone={() => { setDamagedFormOpen(false); load() }} onCancel={() => setDamagedFormOpen(false)} />
        ) : (
          <button
            type="button" onClick={() => setDamagedFormOpen(true)}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:underline"
          >
            <AlertTriangle size={10} /> {t('receiving.flagDamaged')}
          </button>
        )
      )}
      <div className="flex items-center justify-between pt-1">
        <button
          type="button" onClick={() => setReportOpen(true)}
          className="flex items-center gap-1 px-3 py-1.5 rounded-lg dark:bg-white/5 bg-gray-100 text-ink-2 text-2xs font-semibold hover:dark:bg-white/10 hover:bg-gray-200"
        >
          <ClipboardList size={11} /> {t('receiving.generateReport')}
        </button>
        {isApplied ? (
          <span className="flex items-center gap-1 text-2xs font-medium text-success">
            <PackageCheck size={11} /> {t('receiving.completed')}
          </span>
        ) : (
          <button
            type="button" onClick={handleComplete} disabled={completing}
            className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-2xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50"
          >
            <PackageCheck size={11} /> {completing ? t('receiving.completing') : t('receiving.markComplete')}
          </button>
        )}
      </div>
      {reportOpen && (
        <ReceivingReportModal
          discrepancies={detail.discrepancies} invoiceFileName={invoiceFileName}
          onClose={() => setReportOpen(false)}
        />
      )}
    </div>
  )
}

interface ReceivingReportModalProps {
  discrepancies: ReceivingDiscrepancy[]
  invoiceFileName: string
  onClose: () => void
}

function ReceivingReportModal({ discrepancies, invoiceFileName, onClose }: ReceivingReportModalProps) {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation(['procurement', 'common'])
  const describe = (d: ReceivingDiscrepancy) =>
    d.expiry_date ? t('report.expires', { details: d.details, date: formatDate(d.expiry_date) }) : d.details
  const grouped = DISCREPANCY_TYPES.map(type => ({
    type, items: discrepancies.filter(d => d.type === type),
  })).filter(g => g.items.length > 0)

  function handleExportCsv() {
    const header = [t('report.colType'), t('report.colExpected'), t('report.colReceived'), t('report.colStatus'), t('report.colDetails'), t('report.colExpiry')]
    const rows = discrepancies.map(d => [
      discrepancyLabel(d.type), String(d.expected_qty), String(d.received_qty), d.status === 'acknowledged' ? t('report.acknowledged') : t('report.open'), d.details,
      d.expiry_date ? formatDate(d.expiry_date) : '',
    ])
    const csv = [header, ...rows]
      .map(row => row.map(cell => `"${cell.replace(/"/g, '""')}"`).join(','))
      .join('\n')
    const blob = new Blob([csv], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `receiving-report-${invoiceFileName || 'export'}.csv`
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  }

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-lg max-h-[85vh] flex flex-col rounded-xl dark:bg-night-900 bg-white border border-line shadow-xl">
        <div className="flex items-center justify-between px-5 py-4 border-b border-line">
          <h2 className="text-sm font-semibold text-ink">{t('report.title')}</h2>
          <button aria-label={t('common:actions.close')} type="button" onClick={onClose} title={t('common:actions.close')}>
            <X size={16} className="text-ink-4" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-5 space-y-4">
          {grouped.length === 0 ? (
            <p className="text-2xs text-ink-3">{t('report.empty')}</p>
          ) : (
            grouped.map(g => (
              <div key={g.type} className="space-y-1.5">
                <h3 className="text-2xs font-semibold text-ink-2">
                  {discrepancyLabel(g.type)} <span className="dark:text-ink-4 font-normal">({g.items.length})</span>
                </h3>
                <ul className="space-y-1">
                  {g.items.map(d => (
                    <li key={d.id} className="text-2xs text-ink-3 flex items-center justify-between gap-2">
                      <span>{describe(d)}</span>
                      <span className={clsx('shrink-0', d.status === 'acknowledged' ? 'text-green-400' : 'text-amber-400')}>
                        {d.status === 'acknowledged' ? t('report.acknowledged') : t('report.open')}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            ))
          )}
        </div>
        <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-line">
          <button
            type="button" onClick={handleExportCsv}
            className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-fill text-ink-2 text-2xs font-semibold hover:bg-hover"
          >
            <Download size={11} /> {t('report.exportCsv')}
          </button>
        </div>
      </div>
    </Dialog>
  )
}
