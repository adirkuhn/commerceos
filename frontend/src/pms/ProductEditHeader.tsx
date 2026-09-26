import { useState } from 'react'
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
  const [resyncing, setResyncing] = useState(false)
  const [resyncMessage, setResyncMessage] = useState('')

  const handleResync = async () => {
    if (!savedProductId) return
    setResyncing(true)
    setResyncMessage('')
    try {
      await integrationsApi.resyncProductToChannels(token ?? '', savedProductId)
      setResyncMessage('Resync started.')
    } catch (err) {
      setResyncMessage(err instanceof Error ? err.message : 'Failed to resync')
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
          <ArrowLeft size={11} /> Products
        </button>
        <h1 className="text-xl font-semibold text-ink leading-tight">
          {savedProductId ? name || 'Edit product' : 'New product'}
        </h1>
      </div>
      <div className="flex items-center gap-2">
        {resyncMessage && <span className="text-2xs text-ink-4 mr-1">{resyncMessage}</span>}
        {savedProductId && showResync && (
          <button
            type="button" onClick={handleResync} disabled={resyncing}
            title="Force-push this product's current fields and stock to every linked channel, even if nothing changed"
            className="flex items-center gap-1.5 px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
          >
            <RefreshCw size={12} className={resyncing ? 'animate-spin' : undefined} />
            {resyncing ? 'Resyncing…' : 'Resync to channels'}
          </button>
        )}
        {onMerge && (
          <button
            type="button" onClick={onMerge}
            title="This is a duplicate of a product that already exists: move everything into that one and delete this"
            className="flex items-center gap-1.5 px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
          >
            <Merge size={12} /> Merge into…
          </button>
        )}
        <button
          type="button" onClick={onClose}
          className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
        >
          Discard
        </button>
        <button
          type="submit" form="product-general-form" disabled={saving || saveDisabled}
          className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
        >
          {saving ? 'Saving…' : savedProductId ? 'Save' : 'Create product'}
        </button>
      </div>
    </div>
  )
}
