import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { Dialog } from '../../components/common/Dialog'
import type { TemplatePreview } from '../types'

// PreviewDialog shows the email as it will be sent, filled with sample
// data, plus the WhatsApp/SMS text.
export function PreviewDialog({ preview, onClose }: { preview: TemplatePreview; onClose: () => void }) {
  const { t } = useTranslation(['messages', 'common'])
  return (
    <Dialog onClose={onClose} label={t('design.previewTitle')} className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} />
      <div className="relative w-full max-w-3xl h-[85vh] bg-raised rounded-xl border border-line shadow-2xl flex flex-col">
        <div className="flex items-center justify-between px-4 py-3 border-b border-line">
          <div className="min-w-0">
            <p className="text-2xs text-ink-4">{t('design.previewSubject')}</p>
            <p className="text-sm font-medium text-ink truncate">{preview.subject}</p>
          </div>
          <button onClick={onClose} aria-label={t('common:actions.close')}><X size={16} className="text-ink-4" /></button>
        </div>
        <iframe title={t('design.previewTitle')} srcDoc={preview.html} sandbox="" className="flex-1 w-full bg-white" />
        {preview.text && (
          <div className="px-4 py-3 border-t border-line">
            <p className="text-2xs text-ink-4 mb-1">{t('design.previewText')}</p>
            <p className="text-xs text-ink whitespace-pre-wrap">{preview.text}</p>
          </div>
        )}
      </div>
    </Dialog>
  )
}
