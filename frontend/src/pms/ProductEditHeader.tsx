import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Merge, RefreshCw } from 'lucide-react'
import * as integrationsApi from '../integrations/api'

interface ProductEditHeaderProps {
  token: string | null
  savedProductId: string | null
  name: string
  showResync: boolean
  saving: boolean
  saveDisabled: boolean
  onClose: () => void
  // Offered to admins on a saved product only.
  onMerge?: () => void
}

// Title bar of the product editor: back link, title, and the page-level
// actions (resync, discard, save -- save submits the General tab's form).
export function ProductEditHeader({ token, savedProductId, name, showResync, saving, saveDisabled, onClose, onMerge }: ProductEditHeaderProps) {
  const { t } = useTranslation(['pms', 'common'])
  const [resyncing, setResyncing] = useState(false)
  const [resyncMessage, setResyncMessage] = useState('')

  const handleResync = async () => {
    if (!savedProductId) return
    setResyncing(true)
    setResyncMessage('')
    try {
      await integrationsApi.resyncProductToChannels(token ?? '', savedProductId)
      setResyncMessage(t('header.resyncStarted'))
    } catch (err) {
      setResyncMessage(err instanceof Error ? err.message : t('header.resyncFailed'))
    } finally {
      setResyncing(false)
    }
  }

  return (
    <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
      <div>
        <button
          onClick={onClose}
          className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
        >
          <ArrowLeft size={11} /> {t('header.back')}
        </button>
        <h1 className="text-xl font-semibold text-ink leading-tight">
          {savedProductId ? name || t('header.edit') : t('header.new')}
        </h1>
      </div>
      <div className="flex items-center gap-2">
        {resyncMessage && <span className="text-2xs text-ink-4 mr-1">{resyncMessage}</span>}
        {savedProductId && showResync && (
          <button
            type="button" onClick={handleResync} disabled={resyncing}
            title={t('header.resyncHint')}
            className="flex items-center gap-1.5 px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
          >
            <RefreshCw size={12} className={resyncing ? 'animate-spin' : undefined} />
            {resyncing ? t('header.resyncing') : t('header.resync')}
          </button>
        )}
        {onMerge && (
          <button
            type="button" onClick={onMerge}
            title={t('header.mergeHint')}
            className="flex items-center gap-1.5 px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
          >
            <Merge size={12} /> {t('header.merge')}
          </button>
        )}
        <button
          type="button" onClick={onClose}
          className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
        >
          {t('header.discard')}
        </button>
        <button
          type="submit" form="product-general-form" disabled={saving || saveDisabled}
          className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
        >
          {saving ? t('common:actions.saving') : savedProductId ? t('common:actions.save') : t('header.create')}
        </button>
      </div>
    </div>
  )
}
