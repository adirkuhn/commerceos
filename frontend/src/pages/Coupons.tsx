import { useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { Plus, Search, Ticket, Trash2 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useCoupons } from '../coupons/hooks/useCoupons'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useQueryPage, useQueryString } from '../routing/useQueryState'
import { RecordLoading, RecordNotFound } from '../components/common/RecordStatus'
import type { Coupon } from '../coupons/types'
import { useConfirm } from '../components/common/useConfirm'
import { CouponEditPage } from '../coupons/CouponEditPage'
import { useDateFormat } from '../format/DateFormatProvider'
import { apiErrorText } from '../api/apiErrorText'
import { Money } from '../format/amountVisibility'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import { LIST_TOOLBAR, LIST_TOOLBAR_ACTION, LIST_TOOLBAR_INLINE, LIST_TOOLBAR_LEAD, LIST_TOOLBAR_ROW } from '../components/common/listToolbar'

const PAGE_SIZE = 20

// Coupons are tenant-wide, so a fixed amount is in the shop's default
// currency (USD when never set -- the backend's own fallback).
const FALLBACK_CURRENCY = 'USD'

function Discount({ coupon: c, currency }: { coupon: Coupon; currency: string }) {
  const { t } = useTranslation('coupons')
  if (c.discount_type === 'percent') return <>{t('list.percent', { value: (c.discount_value / 100).toFixed(2) })}</>
  return <Trans t={t} i18nKey="list.amountOff" components={{ amount: <Money minor={c.discount_value} currency={currency} /> }} />
}

export default function Coupons() {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation(['coupons', 'common'])
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { coupons, loading, error, create, update, remove } = useCoupons(token ?? '')
  const currency = useTenantSettings(token ?? '').settings?.default_currency || FALLBACK_CURRENCY

  // List view state (search, page) and the open editor (/coupons/new,
  // /coupons/:id) live in the URL -- refresh and Back keep the user in place.
  const [searchParam, setSearch] = useQueryString('q', { resets: ['page'] })
  const search = searchParam ?? ''
  const [page, setPage] = useQueryPage()
  const { editingId, openNew, openEdit, close } = useRoutedEditor('/coupons')
  const [actionError, setActionError] = useState('')

  function handleSearchChange(value: string) {
    setSearch(value)
  }

  const filtered = coupons.filter(c => c.code.toLowerCase().includes(search.toLowerCase()))
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  async function handleDelete(c: Coupon) {
    if (!(await confirm({ title: t('list.deleteTitle', { code: c.code }), message: t('list.deleteMessage'), confirmLabel: t('list.deleteConfirm'), destructive: true }))) return
    setActionError('')
    try {
      await remove(c.id)
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('common:status.error'))
    }
  }

  if (editingId !== null) {
    const editing = editingId === NEW_RECORD ? null : coupons.find(c => c.id === editingId) ?? null
    if (editingId !== NEW_RECORD && !editing) {
      return loading ? <RecordLoading /> : <RecordNotFound label={t('list.recordLabel')} onBack={close} />
    }
    return (
      <CouponEditPage
        key={editingId}
        coupon={editing}
        onCreate={create}
        onUpdate={update}
        onClose={close}
      />
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className={LIST_TOOLBAR}>
        <div className={clsx('flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0', LIST_TOOLBAR_ROW)}>
          <div className="relative flex-1 max-w-sm">
            <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
            <input
              value={search}
              onChange={e => handleSearchChange(e.target.value)}
              placeholder={t('list.search')}
              className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
            />
          </div>
          <div className={clsx('flex items-center gap-2 ml-auto', LIST_TOOLBAR_ACTION)}>
            <button
              onClick={openNew}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover transition-colors font-medium"
            >
              <Plus size={13} /> {t('list.add')}
            </button>
          </div>
        </div>

        <div className={clsx('flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0', LIST_TOOLBAR_ROW)}>
          <span className={clsx('text-xs text-ink-4', LIST_TOOLBAR_LEAD)}>{t('list.count', { count: filtered.length })}</span>
          {(error || actionError) && <span className={clsx('text-xs text-danger ml-auto', LIST_TOOLBAR_INLINE)}>{error || actionError}</span>}
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colCode')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colDiscount')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colUsage')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colExpires')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colStatus')}</th>
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
            {paged.map(c => (
              <tr
                key={c.id}
                onClick={() => openEdit(c.id)}
                className="group hover:bg-hover/60 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded bg-fill flex items-center justify-center">
                    <Ticket size={12} className="text-ink-4" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[200px]">
                  <p className="font-medium font-mono text-ink truncate">{c.code}</p>
                </td>
                <td className="px-3 py-2.5 text-ink-3"><Discount coupon={c} currency={currency} /></td>
                <td className="px-3 py-2.5 text-ink-3">
                  {c.times_used}{c.usage_limit != null ? ` / ${c.usage_limit}` : ''}
                </td>
                <td className="px-3 py-2.5 text-ink-3">{formatDate(c.expires_at)}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-sm text-2xs font-semibold', c.active ? 'bg-success/10 text-success' : 'bg-fill text-ink-4')}>
                    {c.active ? t('common:status.active') : t('common:status.inactive')}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    <button aria-label={t('common:actions.delete')}
                      onClick={e => { e.stopPropagation(); handleDelete(c) }}
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
