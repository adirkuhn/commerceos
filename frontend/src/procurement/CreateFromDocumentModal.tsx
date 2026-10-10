import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import * as procurementApi from './api'
import { Dialog } from '../components/common/Dialog'
import { apiErrorText } from '../api/apiErrorText'

// ── Create-from-document modal ──────────────────────────────────────────────

interface CreateFromDocumentModalProps {
  onClose: () => void
  onCreated: (purchaseOrderId: string, parsingJobId: string) => void
}

// Deliberately just a file picker -- supplier/warehouse/currency/expected
// date were removable friction: supplier and line items both come from the
// same document parse (no need to pick a supplier up front, the PO starts
// needs_review with a suggestion if one can't be matched), warehouse is
// never needed until receiving, and currency defaults from the tenant's
// own system-settings default.
export function CreateFromDocumentModal({ onClose, onCreated }: CreateFromDocumentModalProps) {
  const { token } = useAuth()
  const { settings: tenantSettings } = useTenantSettings(token ?? '')
  const [file, setFile] = useState<File | null>(null)
  const [status, setStatus] = useState<'idle' | 'uploading'>('idle')
  const [formError, setFormError] = useState('')
  const { t } = useTranslation(['procurement', 'common'])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!file) return
    setFormError('')
    setStatus('uploading')
    try {
      const job = await procurementApi.createPurchaseOrderFromDocument(token ?? '', {
        currency_code: tenantSettings?.default_currency || 'USD', file,
      })
      // Navigate immediately instead of blocking here on the parse --
      // it runs async in the background (cmd/scheduler) and can take a
      // few seconds; the PO page itself shows the parsing indicator and
      // picks up line items/supplier as soon as it finishes (see
      // LineItemsSection's initialParsingJobId), so there's no need to
      // freeze this modal waiting for it.
      onCreated(job.purchase_order_id, job.id)
    } catch (err) {
      setFormError(apiErrorText(err) ?? t('upload.failed'))
      setStatus('idle')
    }
  }

  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-md rounded-xl bg-surface border border-line shadow-xl">
        <div className="flex items-center justify-between px-5 py-4 border-b border-line">
          <h2 className="text-sm font-semibold text-ink">{t('upload.title')}</h2>
          <button aria-label={t('common:actions.close')} type="button" onClick={onClose} title={t('common:actions.close')}>
            <X size={16} className="text-ink-4" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          <p className="text-2xs text-ink-4">
            {t('upload.hint')}
          </p>
          <div>
            <label htmlFor="upload-file" className={labelClass}>{t('upload.document')}</label>
            <input id="upload-file" type="file" accept=".pdf,.csv,.xlsx"
              disabled={status !== 'idle'}
              onChange={e => setFile(e.target.files?.[0] ?? null)}
              className="w-full text-xs text-ink-2 file:mr-3 file:px-3 file:py-1.5 file:rounded-lg file:border-0 file:bg-accent-solid file:text-white file:text-xs file:font-medium file:cursor-pointer disabled:opacity-50" />
          </div>
          {formError && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>}
          <div className="flex items-center justify-end gap-2 pt-1">
            <button type="button" onClick={onClose}
              className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors">
              {t('common:actions.cancel')}
            </button>
            <button type="submit" disabled={status !== 'idle' || !file}
              className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors">
              {status === 'uploading' ? t('upload.uploading') : t('upload.submit')}
            </button>
          </div>
        </form>
      </div>
    </Dialog>
  )
}
