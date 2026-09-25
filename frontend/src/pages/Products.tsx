import { useState, useEffect } from 'react'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useQueryPage, useQueryString } from '../routing/useQueryState'
import { Plus, Search, Package, Trash2, Edit3, Copy } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useProducts } from '../pms/hooks/useProducts'
import { useCategories } from '../pms/hooks/useCategories'
import * as pmsApi from '../pms/api'
import { useConfirm } from '../components/common/useConfirm'
import { STATUS_STYLES, formatPrice } from '../pms/productFormat'
import { ProductEditPage } from '../pms/ProductEditPage'
const PAGE_SIZE = 20

export default function Products() {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()

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
      setDuplicateError(err instanceof Error ? err.message : 'Failed to duplicate product')
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
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={searchInput}
            onChange={e => setSearchInput(e.target.value)}
            placeholder="Search by name or slug..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <button
            onClick={openNew}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover transition-colors font-medium"
          >
            <Plus size={13} /> New product
          </button>
        </div>
      </div>

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{total} product{total === 1 ? '' : 's'}</span>
        {(error || duplicateError) && <span className="text-xs text-danger ml-auto">{error || duplicateError}</span>}
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Product</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Category</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Price</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Sale price</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 w-8" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">Loading…</td></tr>
            )}
            {!loading && products.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">No products yet</td></tr>
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
                  {p.price_minor_units != null ? formatPrice(p.price_minor_units, p.currency_code ?? '') : '—'}
                </td>
                <td className="px-3 py-2.5">
                  {p.sale_price_minor_units != null ? (
                    <span className="text-danger font-medium">{formatPrice(p.sale_price_minor_units, p.currency_code ?? '')}</span>
                  ) : (
                    <span className="text-ink-4">—</span>
                  )}
                </td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium capitalize', STATUS_STYLES[p.status])}>
                    {p.status}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    <button aria-label="Edit"
                      onClick={e => { e.stopPropagation(); openEditor(p.id) }}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title="Edit"
                    >
                      <Edit3 size={12} className="text-ink-3" />
                    </button>
                    <button aria-label="Duplicate"
                      onClick={e => { e.stopPropagation(); handleDuplicate(p.id) }}
                      disabled={duplicatingId === p.id}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors disabled:opacity-50"
                      title="Duplicate"
                    >
                      <Copy size={12} className="text-ink-3" />
                    </button>
                    <button aria-label="Delete"
                      onClick={async e => {
                        e.stopPropagation()
                        if (await confirm({ title: `Delete "${p.name}"?`, message: 'Its variants and SKUs are deleted too.', confirmLabel: 'Delete product', destructive: true })) {
                          removeProduct(p.id)
                        }
                      }}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title="Delete"
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
