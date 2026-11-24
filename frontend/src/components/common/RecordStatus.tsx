import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'

// Placeholders for a page whose record comes from the URL
// (/coupons/:id, ...): shown while the list that holds it is loading, and
// when the id matches nothing (deleted record, stale bookmark) -- instead of
// silently rendering an empty "create" form for a URL that meant "edit".

export function RecordLoading() {
  const { t } = useTranslation(['ui', 'common'])
  return <div className="flex h-full items-center justify-center text-sm text-ink-4">{t('common:status.loading')}</div>
}

export function RecordNotFound({ label, onBack }: { label: string; onBack: () => void }) {
  const { t } = useTranslation(['ui', 'common'])
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3">
      <p className="text-sm font-medium text-ink">{t('record.notFound', { label })}</p>
      <p className="text-xs text-ink-4">{t('record.notFoundHint')}</p>
      <button
        type="button" onClick={onBack}
        className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-fill text-ink-2"
      >
        <ArrowLeft size={13} /> {t('record.backToList')}
      </button>
    </div>
  )
}
