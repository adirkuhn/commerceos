import { useState } from 'react'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Dialog } from '../../components/common/Dialog'
import { apiErrorText } from '../../api/apiErrorText'
import type { EmailSupplierIssuesInput, EmailSupplierIssuesResult, SupplierIssueRef } from '../types'

interface EmailSupplierDialogProps {
  // The supplier's address; empty when it has none.
  initialTo: string
  initialSubject: string
  initialBody: string
  issues: SupplierIssueRef[]
  onSend: (input: EmailSupplierIssuesInput) => Promise<EmailSupplierIssuesResult>
  onSent: (result: EmailSupplierIssuesResult) => void
  onClose: () => void
}

const FIELD = 'w-full rounded-lg border border-line bg-raised px-3 py-2 text-xs text-ink'

// The message is prefilled from the issue list and stays editable: it goes
// out as typed (plain text).
export function EmailSupplierDialog({ initialTo, initialSubject, initialBody, issues, onSend, onSent, onClose }: EmailSupplierDialogProps) {
  const { t } = useTranslation(['procurement', 'common'])
  const [to, setTo] = useState(initialTo)
  const [subject, setSubject] = useState(initialSubject)
  const [body, setBody] = useState(initialBody)
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')
  const complete = to.trim() !== '' && subject.trim() !== '' && body.trim() !== ''

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSending(true)
    setError('')
    try {
      onSent(await onSend({ to: to.trim(), subject: subject.trim(), body, issues }))
    } catch (err) {
      setError(apiErrorText(err) ?? t('supplierIssues.email.failed'))
      setSending(false)
    }
  }

  return (
    <Dialog onClose={onClose} dismissible={!sending} className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <form onSubmit={handleSubmit} className="w-full max-w-xl max-h-[90vh] flex flex-col rounded-xl dark:bg-night-900 bg-white border border-line shadow-xl">
        <div className="flex items-center justify-between px-5 py-4 border-b border-line">
          <h2 className="text-sm font-semibold text-ink">{t('supplierIssues.email.title')}</h2>
          <button type="button" onClick={onClose} disabled={sending} aria-label={t('common:actions.close')} title={t('common:actions.close')}>
            <X size={16} className="text-ink-4" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-5 space-y-3">
          <label className="block space-y-1">
            <span className="text-2xs font-medium text-ink-3">{t('supplierIssues.email.to')}</span>
            <input type="email" required value={to} onChange={e => setTo(e.target.value)} className={FIELD} />
          </label>
          {!initialTo && <p className="text-2xs text-warning">{t('supplierIssues.email.noAddress')}</p>}
          <label className="block space-y-1">
            <span className="text-2xs font-medium text-ink-3">{t('supplierIssues.email.subject')}</span>
            <input required maxLength={200} value={subject} onChange={e => setSubject(e.target.value)} className={FIELD} />
          </label>
          <label className="block space-y-1">
            <span className="text-2xs font-medium text-ink-3">{t('supplierIssues.email.body')}</span>
            <textarea required rows={14} maxLength={20000} value={body} onChange={e => setBody(e.target.value)} className={`${FIELD} font-mono leading-relaxed`} />
          </label>
          <p className="text-2xs text-ink-4">{t('supplierIssues.email.claimHint')}</p>
          {error && <p role="alert" className="text-2xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error}</p>}
        </div>
        <div className="flex justify-end gap-2 px-5 py-3 border-t border-line">
          <button type="button" onClick={onClose} disabled={sending} className="px-3 py-1.5 rounded-lg text-xs text-ink-3 hover:bg-hover disabled:opacity-50">
            {t('common:actions.cancel')}
          </button>
          <button type="submit" disabled={!complete || sending} className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-semibold disabled:opacity-50">
            {sending ? t('supplierIssues.email.sending') : t('supplierIssues.email.send')}
          </button>
        </div>
      </form>
    </Dialog>
  )
}
