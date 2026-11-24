import { useState } from 'react'
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import { Plus, Search, User, Trash2, Edit3 } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { LIST_TOOLBAR, LIST_TOOLBAR_ACTION, LIST_TOOLBAR_INLINE, LIST_TOOLBAR_LEAD, LIST_TOOLBAR_ROW } from '../components/common/listToolbar'
import { useCustomers } from '../customers/hooks/useCustomers'
import type { Customer, CustomerInput } from '../customers/types'
import { idFromPath } from '../routes'
import { useQueryPage, useQueryString } from '../routing/useQueryState'
import { useStackNavigation } from '../routing/useStackNavigation'
import { RecordLoading, RecordNotFound } from '../components/common/RecordStatus'
import { useConfirm } from '../components/common/useConfirm'
import { CustomerDetailPage } from '../customers/CustomerDetailPage'
import { CustomerEditPage } from '../customers/CustomerEditPage'
import { customerName } from '../customers/identity/customerIdentity'
import { apiErrorText } from '../api/apiErrorText'

const PAGE_SIZE = 20

// The Notes editor on the detail view only changes customer_note, but
// update() is a full replace (see UpdateCustomerUseCase) -- round-trip every
// other field unchanged rather than adding a second, note-only update path.
// Shipping stays out: sent, it would overwrite the default shipping address
// (left out, the address book is kept).
function customerToInput(c: Customer): CustomerInput {
  return {
    email: c.email, first_name: c.first_name, last_name: c.last_name, username: c.username, role: c.role,
    billing: c.billing, avatar_url: c.avatar_url,
    customer_note: c.customer_note,
  }
}

export default function Customers() {
  const { t } = useTranslation(['customers', 'common'])
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { customers, loading, error, create, update, remove } = useCustomers(token ?? '')

  // Search and page live in the URL (?q=, ?page=) -- refresh and Back keep them.
  const [searchParam, setSearch] = useQueryString('q', { resets: ['page'] })
  const search = searchParam ?? ''
  const [page, setPage] = useQueryPage()
  // The URL is the source of truth for "which customer is open"
  // (/customers/<uuid>, or /customers/new) -- survives a refresh, and is how
  // global search (Cmd+K) deep-links straight to a specific customer. Same
  // convention as Products.tsx's editingId (see its comment for why this
  // reads location.pathname on every render instead of useParams).
  //
  // /customers/<id> alone opens the read-only detail view (Amount
  // spent/Orders, Timeline, Notes, addresses); ?edit=1 on top of that opens
  // the edit form.
  const location = useLocation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const openId = idFromPath(location.pathname, '/customers')
  const isEditing = searchParams.get('edit') === '1'
  const { open, exitTo, exitToOpener } = useStackNavigation()
  const openEditor = (id: string) => open(`/customers/${id}?edit=1`)
  // Back to the exact list URL (search/page) it was opened from.
  const closeEditor = () => exitToOpener('/customers')
  const [deleteError, setDeleteError] = useState('')

  function handleSearchChange(value: string) {
    setSearch(value)
  }

  const filtered = customers.filter(c =>
    [customerName(c), c.email, c.client_number ?? '', c.kvk_number ?? ''].some(v => v.toLowerCase().includes(search.toLowerCase())),
  )
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  async function handleDelete(c: Customer) {
    if (!(await confirm({ title: t('list.deleteTitle', { name: customerName(c) }), message: t('list.deleteMessage'), confirmLabel: t('list.deleteConfirm'), destructive: true }))) return
    setDeleteError('')
    try {
      await remove(c.id)
    } catch (err) {
      setDeleteError(apiErrorText(err) ?? t('errors.unknown'))
    }
  }

  if (openId === 'new') {
    return <CustomerEditPage customer={null} onCreate={create} onUpdate={update} onClose={closeEditor} />
  }
  if (openId !== null) {
    const existing = customers.find(c => c.id === openId) ?? null
    if (isEditing) {
      if (!existing) {
        return loading ? <RecordLoading /> : <RecordNotFound label={t('list.recordLabel')} onBack={closeEditor} />
      }
      return (
        <CustomerEditPage
          customer={existing}
          onCreate={create}
          onUpdate={update}
          onClose={() => exitTo(`/customers/${openId}`)}
        />
      )
    }
    return (
      <CustomerDetailPage
        customer={existing}
        loading={loading}
        token={token ?? ''}
        onEdit={() => openEditor(openId)}
        onClose={closeEditor}
        onSaveNote={note => update(openId, { ...customerToInput(existing!), customer_note: note })}
      />
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar -- rows that flow into one band (common/listToolbar) */}
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
              onClick={() => navigate('/customers/new')}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover transition-colors font-medium"
            >
              <Plus size={13} /> {t('list.add')}
            </button>
          </div>
        </div>

        {/* Stats strip */}
        <div className={clsx('flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0', LIST_TOOLBAR_ROW)}>
          <span className={clsx('text-xs text-ink-4', LIST_TOOLBAR_LEAD)}>{t('list.count', { count: filtered.length })}</span>
          {(error || deleteError) && <span className={clsx('text-xs text-danger ml-auto', LIST_TOOLBAR_INLINE)}>{error || deleteError}</span>}
        </div>
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('common:fields.customer')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colLocation')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('common:fields.status')}</th>
              <th className="px-3 py-2.5 w-16" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={5} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={5} className="px-4 py-6 text-center text-ink-4">{t('list.empty')}</td></tr>
            )}
            {paged.map(c => (
              <tr
                key={c.id}
                onClick={() => navigate(`/customers/${c.id}`)}
                className="group hover:bg-hover/60 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-night-700 bg-gray-200 flex items-center justify-center">
                    <User size={12} className="text-ink-4" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[280px]">
                  <p className="font-medium text-ink truncate">
                    {customerName(c)}
                    {c.client_number && <span className="ml-1.5 text-2xs font-normal text-ink-4 tabular-nums">#{c.client_number}</span>}
                  </p>
                  <p className="text-2xs text-ink-4 truncate">
                    {c.customer_type === 'company' && <span className="mr-1.5">{t('identity.types.company')}</span>}
                    {c.email}
                  </p>
                </td>
                <td className="px-3 py-2.5 text-ink-3">
                  {[c.billing?.city, c.billing?.country].filter(Boolean).join(', ') || '—'}
                </td>
                <td className="px-3 py-2.5">
                  <span className={clsx(
                    'px-2 py-0.5 rounded-full text-2xs font-medium',
                    c.is_paying_customer ? 'bg-green-500/15 text-success' : 'bg-gray-500/15 text-ink-4',
                  )}>
                    {c.is_paying_customer ? t('list.paying') : t('list.noOrders')}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    <button aria-label={t('common:actions.edit')}
                      onClick={e => { e.stopPropagation(); openEditor(c.id) }}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title={t('common:actions.edit')}
                    >
                      <Edit3 size={12} className="text-ink-3" />
                    </button>
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
