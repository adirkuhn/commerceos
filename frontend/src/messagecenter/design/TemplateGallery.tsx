import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FilePlus, Mail, MessageSquare } from 'lucide-react'
import { useAuth } from '../../auth/AuthContext'
import { apiErrorText } from '../../api/apiErrorText'
import * as api from '../api'
import type { MessageTemplateDefault } from '../types'
import { triggerLabels } from '../triggerLabels'
import { plainTextExcerpt } from './designHelpers'

interface Props {
  onBlank: () => void
  onPick: (d: MessageTemplateDefault) => Promise<void>
}

// TemplateGallery is the first step of "New template": start blank, or
// from one of the built-in defaults (in the store's language) that the
// tenant then customizes.
export function TemplateGallery({ onBlank, onPick }: Props) {
  const { token } = useAuth()
  const { t } = useTranslation(['messages', 'common'])
  const [defaults, setDefaults] = useState<MessageTemplateDefault[] | null>(null)
  const [busyKey, setBusyKey] = useState('')
  const [error, setError] = useState('')
  const labels = triggerLabels(t)

  useEffect(() => {
    api.listMessageTemplateDefaults(token ?? '').then(setDefaults, err => {
      setDefaults([])
      setError(apiErrorText(err) ?? t('common:status.error'))
    })
  }, [token, t])

  async function pick(d: MessageTemplateDefault) {
    setBusyKey(d.key)
    setError('')
    try {
      await onPick(d)
    } catch (err) {
      setError(apiErrorText(err) ?? t('design.gallery.createFailed'))
      setBusyKey('')
    }
  }

  const card = 'text-left p-3 rounded-lg border border-line bg-surface hover:bg-hover disabled:opacity-50 transition-colors'
  return (
    <div className="space-y-3">
      <p className="text-xs text-ink-3">{t('design.gallery.hint')}</p>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-[60vh] overflow-y-auto">
        <button type="button" onClick={onBlank} disabled={busyKey !== ''} className={card}>
          <span className="flex items-center gap-1.5 text-xs font-medium text-ink"><FilePlus size={13} /> {t('design.gallery.blank')}</span>
          <span className="block text-2xs text-ink-4 mt-1">{t('design.gallery.blankHint')}</span>
        </button>
        {defaults === null && <p className="text-xs text-ink-4 p-3">{t('common:status.loading')}</p>}
        {defaults?.map(d => (
          <button key={d.key} type="button" onClick={() => pick(d)} disabled={busyKey !== ''} className={card}>
            <span className="flex items-center gap-1.5 text-xs font-medium text-ink">
              {d.name}
              {d.channels.includes('email') && <Mail size={11} className="text-ink-4" />}
              {d.channels.includes('whatsapp') && <MessageSquare size={11} className="text-ink-4" />}
            </span>
            <span className="block text-2xs text-ink-4 mt-0.5">{t('design.gallery.sentOn', { trigger: labels[d.trigger_subject] ?? d.trigger_subject })}</span>
            <span className="block text-2xs text-ink-3 mt-1 line-clamp-2">
              {busyKey === d.key ? t('common:status.loading') : plainTextExcerpt(d.text_body || d.body)}
            </span>
          </button>
        ))}
      </div>
      {error && <p role="alert" className="text-xs text-danger">{error}</p>}
    </div>
  )
}
