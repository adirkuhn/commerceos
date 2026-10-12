import { useTranslation } from 'react-i18next'
import { ArrowLeft, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useAuth } from '../../auth/AuthContext'
import { apiErrorText } from '../../api/apiErrorText'
import { useDateFormat } from '../../format/DateFormatProvider'
import { useConfirm } from '../../components/common/useConfirm'
import { useResource } from '../../routing/useResource'
import { triggerLabels } from '../triggerLabels'
import type { TriggerSubject } from '../types'
import * as api from '../api'
import { STATUS_CLASSES, sentReason, sentSourceLabels, sentStatusLabels } from './sentLabels'

interface Props {
  id: string
  onBack: () => void
  onDeleted: () => void
}

// SentEmailDetail shows one email from the history as it was sent.
export function SentEmailDetail({ id, onBack, onDeleted }: Props) {
  const { token } = useAuth()
  const { t } = useTranslation(['messages', 'common'])
  const { formatDateTime } = useDateFormat()
  const { confirm, confirmDialog } = useConfirm()
  const { data: email, error, loading } = useResource(() => api.getSentEmail(token ?? '', id), [token, id])
  const [deleteError, setDeleteError] = useState('')

  async function remove() {
    if (!(await confirm({ title: t('sent.deleteTitle', { count: 1 }), message: t('sent.deleteMessage'), confirmLabel: t('sent.deleteConfirm'), destructive: true }))) return
    try {
      await api.deleteSentEmail(token ?? '', id)
      onDeleted()
    } catch (err) {
      setDeleteError(apiErrorText(err) ?? t('common:status.error'))
    }
  }

  const triggers = triggerLabels(t)
  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 px-5 py-2.5 border-b border-line bg-sunken shrink-0">
        <button onClick={onBack} aria-label={t('common:actions.back')} className="p-1 rounded hover:bg-hover">
          <ArrowLeft size={16} className="text-ink-3" />
        </button>
        <h2 className="text-sm font-semibold text-ink truncate">{email?.subject ?? ''}</h2>
        {email && (
          <button onClick={remove} className="ml-auto flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg text-danger hover:bg-hover">
            <Trash2 size={13} /> {t('common:actions.delete')}
          </button>
        )}
      </div>
      {loading && <p className="p-5 text-xs text-ink-4">{t('common:status.loading')}</p>}
      {(error || deleteError) && <p role="alert" className="p-5 text-xs text-danger">{error || deleteError}</p>}
      {email && (
        <div className="flex flex-col flex-1 min-h-0">
          <dl className="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1 px-5 py-3 text-xs border-b border-line">
            <dt className="text-ink-4">{t('sent.colDate')}</dt><dd className="text-ink-2">{formatDateTime(email.created_at)}</dd>
            <dt className="text-ink-4">{t('sent.colRecipient')}</dt><dd className="text-ink-2">{email.recipient || '—'}</dd>
            <dt className="text-ink-4">{t('sent.colSource')}</dt>
            <dd className="text-ink-2">
              {sentSourceLabels(t)[email.source]}
              {email.trigger_subject && ` · ${triggers[email.trigger_subject as TriggerSubject] ?? email.trigger_subject}`}
            </dd>
            <dt className="text-ink-4">{t('sent.colStatus')}</dt>
            <dd>
              <span className={STATUS_CLASSES[email.status]}>{sentStatusLabels(t)[email.status]}</span>
              {email.status !== 'sent' && <span className="text-ink-3"> · {sentReason(t, email)}</span>}
            </dd>
          </dl>
          {email.html_body
            ? <iframe title={t('sent.bodyTitle')} srcDoc={email.html_body} sandbox="" className="flex-1 w-full bg-white" />
            : <p className="p-5 text-xs text-ink-4">{t('sent.noBody')}</p>}
        </div>
      )}
      {confirmDialog}
    </div>
  )
}
