import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

interface PaginationProps {
  page: number
  pageSize: number
  totalItems: number
  onPageChange: (page: number) => void
}

export default function Pagination({ page, pageSize, totalItems, onPageChange }: PaginationProps) {
  const { t } = useTranslation(['ui', 'common'])
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize))
  if (totalPages <= 1) return null

  const from = totalItems === 0 ? 0 : (page - 1) * pageSize + 1
  const to = Math.min(page * pageSize, totalItems)

  return (
    <div className="flex items-center justify-between px-5 py-2.5 border-t border-line bg-sunken shrink-0">
      <span className="text-2xs text-ink-4">
        {t('common:pagination.range', { from, to, total: totalItems })}
      </span>
      <div className="flex items-center gap-1">
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          aria-label={t('common:pagination.previousPage')}
          className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        >
          <ChevronLeft size={13} className="text-ink-3" />
        </button>
        <span className="text-2xs text-ink-3 px-1.5">
          {t('ui:pagination.pageOf', { page, total: totalPages })}
        </span>
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
          aria-label={t('common:pagination.nextPage')}
          className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        >
          <ChevronRight size={13} className="text-ink-3" />
        </button>
      </div>
    </div>
  )
}
