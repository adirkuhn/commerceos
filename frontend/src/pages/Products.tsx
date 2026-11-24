import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useQueryPage, useQueryString } from '../routing/useQueryState'
import { Plus, Search, Package, Trash2, Edit3, Copy } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { LIST_TOOLBAR, LIST_TOOLBAR_ACTION, LIST_TOOLBAR_INLINE, LIST_TOOLBAR_LEAD, LIST_TOOLBAR_ROW } from '../components/common/listToolbar'
import { useProducts } from '../pms/hooks/useProducts'
import { useCategories } from '../pms/hooks/useCategories'
import * as pmsApi from '../pms/api'
import { CodedApiError } from '../api/codedApiError'
import type { Product, UpdateProductPayload } from '../pms/types'
import { useConfirm } from '../components/common/useConfirm'
import { STATUS_STYLES } from '../pms/productFormat'
import { Money } from '../format/amountVisibility'
import { ProductEditPage } from '../pms/ProductEditPage'
import { apiErrorText } from '../api/apiErrorText'
const PAGE_SIZE = 20

// 409 codes for "can't delete, history references it" (backend
// domain/pms ErrProductHasHistory and its specific kinds).
const PRODUCT_HISTORY_CODES = new Set(['PRODUCT_HAS_HISTORY', 'PRODUCT_HAS_RECEIVING_HISTORY', 'PRODUCT_HAS_SALES_ORDER_HISTORY', 'PRODUCT_HAS_PICKING_HISTORY', 'PRODUCT_HAS_DISCARD_HISTORY'])

const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1)

// UpdateProduct is a full replace -- everything but status stays as is.
function archivedPayload(p: Product): UpdateProductPayload {
  return {
    name: p.name, description: p.description, category_id: p.category_id ?? null, status: 'archived',
    has_variants: p.has_variants, sync_variants_as_woo_variations: p.sync_variants_as_woo_variations, tags: p.tags,
  }
}

export default function Products() {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { t } = useTranslation(['pms', 'common'])

  // Search (?q=), page (?page=) and the open product (/products/<id>, or
  // /products/new) all live in the URL -- survives a refresh, is a real
  // shareable link, and Back/closing the editor returns to the same list
  // page. It also doubles as the "re-clicking Products in the sidebar
  // returns to the list" fix: Sidebar's setActiveNav navigates to the plain
  // "/products" path (see AppContext), which has no id segment, so
  // editingId goes back to null on its own.
  //
  // Read from the URL on every render (useRoutedEditor uses useLocation,
  // not useParams) so this reflects navigation that doesn't remount the
  // page -- e.g. global search's CommandPalette navigating straight to
  // "/products/<id>" while Products is already mounted.
  const [urlSearch, setUrlSearch] = useQueryString('q', { resets: ['page'] })
  const search = urlSearch ?? ''
  const [page, setPage] = useQueryPage()
  const { editingId, openEdit: openEditor, openNew, close: closeEditor, showSaved } = useRoutedEditor('/products')

  // The box debounces into ?q= so the server isn't hit on every keystroke.
  // Only writes when the typed value actually differs from the URL: writing
  // on mount too would reset ?page= (the search write clears it) for a
  // page the user landed on directly.
  const [searchInput, setSearchInput] = useState(search)
  useEffect(() => {
    if (searchInput === search) return
    const timeout = setTimeout(() => setUrlSearch(searchInput), 300)
    return () => clearTimeout(timeout)
  }, [searchInput, search, setUrlSearch])
  // Back/Forward to a URL with a different ?q= updates the box too.
  useEffect(() => { setSearchInput(search) }, [search])

  const { products, total, loading, error, create: createProduct, update: updateProduct, remove: removeProduct } =
    useProducts(token ?? '', { page, perPage: PAGE_SIZE, search })
  const { categories, create: createCategory, update: updateCategory } = useCategories(token ?? '')

  const categoryName = (id?: string | null) => categories.find(c => c.id === id)?.name ?? '—'

  // A product whose SKUs have receiving/sales/picking/discard history can't
  // be deleted (409 PRODUCT_HAS_*_HISTORY) -- archiving keeps that history
  // and takes it out of the active catalog, so it's offered instead.
  const [deleteError, setDeleteError] = useState('')
  const handleDelete = async (p: Product) => {
    setDeleteError('')
    try {
      await removeProduct(p.id)
    } catch (err) {
      if (!(err instanceof CodedApiError && PRODUCT_HISTORY_CODES.has(err.code))) {
        setDeleteError(apiErrorText(err) ?? t('list.deleteFailed'))
        return
      }
      const archive = await confirm({
        title: t('list.cantDeleteTitle', { name: p.name }),
        message: t('list.cantDeleteMessage', { reason: apiErrorText(err) ?? `${capitalize(err.message)}.` }),
        confirmLabel: t('list.archive'),
      })
      if (!archive) return
      try {
        await updateProduct(p.id, archivedPayload(p))
      } catch (archiveErr) {
        setDeleteError(apiErrorText(archiveErr) ?? t('list.archiveFailed'))
      }
    }
  }

  const [duplicatingId, setDuplicatingId] = useState<string | null>(null)
  const [duplicateError, setDuplicateError] = useState('')
  const handleDuplicate = async (id: string) => {
    if (!token) return
    setDuplicatingId(id)
    setDuplicateError('')
    try {
      const detail = await pmsApi.duplicateProduct(token, id)
      openEditor(detail.product.id)
    } catch (err) {
      setDuplicateError(apiErrorText(err) ?? t('list.duplicateFailed'))
    } finally {
      setDuplicatingId(null)
    }
  }

  // Swaps /products/new for the real id the instant creation succeeds --
  // without this, a refresh in the gap between "create" and the next
  // explicit navigation would reopen a blank "New product" form instead of
  // the product that was just created. Replace, not push: Back shouldn't
  // land on the blank create form either.
  const handleCreateProduct = async (payload: Parameters<typeof createProduct>[0]) => {
    const created = await createProduct(payload)
    showSaved(created.id)
    return created
  }

  if (editingId !== null) {
    return (
      <ProductEditPage
        productId={editingId === NEW_RECORD ? null : editingId}
        categories={categories}
        onCreateCategory={createCategory}
        onUpdateCategory={updateCategory}
        onCreate={handleCreateProduct}
        onUpdate={updateProduct}
        onClose={closeEditor}
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
              value={searchInput}
              onChange={e => setSearchInput(e.target.value)}
              placeholder={t('list.search')}
              className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
            />
          </div>
          <div className={clsx('flex items-center gap-2 ml-auto', LIST_TOOLBAR_ACTION)}>
            <button
              onClick={openNew}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover transition-colors font-medium"
            >
              <Plus size={13} /> {t('list.new')}
            </button>
          </div>
        </div>

        {/* Stats strip */}
        <div className={clsx('flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0', LIST_TOOLBAR_ROW)}>
          <span className={clsx('text-xs text-ink-4', LIST_TOOLBAR_LEAD)}>{t('list.count', { count: total })}</span>
          {(error || duplicateError || deleteError) && <span className={clsx('text-xs text-danger ml-auto', LIST_TOOLBAR_INLINE)}>{error || duplicateError || deleteError}</span>}
        </div>
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colProduct')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colCategory')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colPrice')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colSalePrice')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colStatus')}</th>
              <th className="px-3 py-2.5 w-8" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>
            )}
            {!loading && products.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">{t('list.empty')}</td></tr>
            )}
            {products.map(p => (
              <tr
                key={p.id}
                onClick={() => openEditor(p.id)}
                className="group hover:bg-hover/60 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  {p.image_url ? (
                    <img src={p.image_url} alt="" className="w-6 h-6 rounded object-cover dark:bg-night-700 bg-gray-200" />
                  ) : (
                    <div className="w-6 h-6 rounded dark:bg-night-700 bg-gray-200 flex items-center justify-center">
                      <Package size={12} className="text-ink-4" />
                    </div>
                  )}
                </td>
                <td className="px-3 py-2.5 max-w-[220px]">
                  <p className="font-medium text-ink truncate">{p.name}</p>
                  <p className="text-2xs font-mono text-ink-4 truncate">{p.slug}</p>
                </td>
                <td className="px-3 py-2.5 text-ink-3">{categoryName(p.category_id)}</td>
                <td className="px-3 py-2.5 text-ink-2">
                  {p.price_minor_units != null ? <Money minor={p.price_minor_units} currency={p.currency_code ?? ''} /> : '—'}
                </td>
                <td className="px-3 py-2.5">
                  {p.sale_price_minor_units != null ? (
                    <span className="text-danger font-medium"><Money minor={p.sale_price_minor_units} currency={p.currency_code ?? ''} /></span>
                  ) : (
                    <span className="text-ink-4">—</span>
                  )}
                </td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium capitalize', STATUS_STYLES[p.status])}>
                    {{ draft: t('statusTag.draft'), active: t('statusTag.active'), archived: t('statusTag.archived') }[p.status] ?? p.status}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    <button aria-label={t('common:actions.edit')}
                      onClick={e => { e.stopPropagation(); openEditor(p.id) }}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title={t('common:actions.edit')}
                    >
                      <Edit3 size={12} className="text-ink-3" />
                    </button>
                    <button aria-label={t('list.duplicate')}
                      onClick={e => { e.stopPropagation(); handleDuplicate(p.id) }}
                      disabled={duplicatingId === p.id}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors disabled:opacity-50"
                      title={t('list.duplicate')}
                    >
                      <Copy size={12} className="text-ink-3" />
                    </button>
                    <button aria-label={t('common:actions.delete')}
                      onClick={async e => {
                        e.stopPropagation()
                        if (await confirm({ title: t('list.deleteTitle', { name: p.name }), message: t('list.deleteMessage'), confirmLabel: t('list.deleteConfirm'), destructive: true })) {
                          await handleDelete(p)
                        }
                      }}
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

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
      {confirmDialog}
    </div>
  )
}
