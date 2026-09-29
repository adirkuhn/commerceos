import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Search, Tag, Trash2 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useQueryPage, useQueryString } from '../routing/useQueryState'
import { usePromotions } from '../promotions/hooks/usePromotions'
import type { Promotion, PromotionStatus } from '../promotions/types'
import { useConfirm } from '../components/common/useConfirm'
import { PromotionEditPage } from '../promotions/PromotionEditPage'
import { useDateFormat } from '../format/DateFormatProvider'

const PAGE_SIZE = 20

const STATUS_STYLE: Record<PromotionStatus, string> = {
  draft: 'bg-gray-500/15 text-ink-4',
  active: 'bg-green-500/15 text-success',
  ended: 'bg-gray-500/15 text-ink-4',
}

export default function Promotions() {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation(['promotions', 'common'])
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { promotions, loading, error, create, update, remove } = usePromotions(token ?? '')

  // Search, page and the open editor (/promotions/new, /promotions/:id) live in the
  // URL -- refresh and Back keep the user in place.
  const [searchParam, setSearch] = useQueryString('q', { resets: ['page'] })
  const search = searchParam ?? ''
  const [page, setPage] = useQueryPage()
  const { editingId, openNew, openEdit, close } = useRoutedEditor('/promotions')
  const [deleteError, setDeleteError] = useState('')

  function handleSearchChange(value: string) {
    setSearch(value)
  }

  const filtered = promotions.filter(p => p.name.toLowerCase().includes(search.toLowerCase()))
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  async function handleDelete(p: Promotion) {
    if (!(await confirm({ title: t('list.deleteTitle', { name: p.name }), message: t('list.deleteMessage'), confirmLabel: t('list.deleteConfirm'), destructive: true }))) return
    setDeleteError('')
    try {
      await remove(p.id)
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : t('common:status.error'))
    }
  }

  if (editingId !== null) {
    return (
      <PromotionEditPage
        key={editingId}
        token={token ?? ''}
        promotionId={editingId === NEW_RECORD ? null : editingId}
        onCreate={create}
        onUpdate={update}
        onClose={close}
      />
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={search}
            onChange={e => handleSearchChange(e.target.value)}
            placeholder={t('list.search')}
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <button
            onClick={openNew}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover transition-colors font-medium"
          >
            <Plus size={13} /> {t('list.add')}
          </button>
        </div>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{t('list.count', { count: filtered.length })}</span>
        {(error || deleteError) && <span className="text-xs text-danger ml-auto">{error || deleteError}</span>}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('list.colPromotion')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('list.colType')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('list.colStarts')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('list.colEnds')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('list.colStatus')}</th>
              <th className="px-3 py-2.5 w-10" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">{t('list.empty')}</td></tr>
            )}
            {paged.map(p => (
              <tr
                key={p.id}
                onClick={() => openEdit(p.id)}
                className="group hover:bg-hover/60 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-night-700 bg-gray-200 flex items-center justify-center">
                    <Tag size={12} className="text-ink-4" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[280px]">
                  <p className="font-medium text-ink truncate">{p.name}</p>
                </td>
                <td className="px-3 py-2.5 text-ink-3">{p.type === 'fixed_amount' ? t('type.fixed_amount') : t('type.percentage')}</td>
                <td className="px-3 py-2.5 text-ink-3">{formatDate(p.starts_at)}</td>
                <td className="px-3 py-2.5 text-ink-3">{formatDate(p.ends_at)}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium capitalize', STATUS_STYLE[p.status])}>
                    {{ draft: t('status.draft'), active: t('status.active'), ended: t('status.ended') }[p.status] ?? p.status}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    <button aria-label={t('common:actions.delete')}
                      onClick={e => { e.stopPropagation(); handleDelete(p) }}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title={t('common:actions.delete')}
                    >
                      <Trash2 size={12} className="text-ink-3 hover:text-danger" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={filtered.length} onPageChange={setPage} />
      {confirmDialog}
    </div>
  )
}
