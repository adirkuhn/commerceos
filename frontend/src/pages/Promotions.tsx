import { useEffect, useState } from 'react'
import { Plus, Search, Tag, Trash2, ArrowLeft } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useQueryPage, useQueryString } from '../routing/useQueryState'
import { usePromotions } from '../promotions/hooks/usePromotions'
import { getPromotion } from '../promotions/api'
import { resolveBarcode } from '../pms/api'
import { getPromotionPerformanceReport } from '../reports/api'
import type { Promotion, PromotionInput, PromotionLine, PromotionStatus, PromotionType, CreatePromotionResult } from '../promotions/types'
import type { PromotionPerformanceReport } from '../reports/types'
import { useConfirm } from '../components/common/useConfirm'
import { ProductSkuPicker } from '../components/common/ProductSkuPicker'
import { ClearanceBatchSelect } from '../promotions/components/ClearanceBatchSelect'
import type { SKU } from '../pms/types'

const PAGE_SIZE = 20

const STATUS_STYLE: Record<PromotionStatus, string> = {
  draft: 'bg-gray-500/15 text-ink-4',
  active: 'bg-green-500/15 text-success',
  ended: 'bg-gray-500/15 text-ink-4',
}

function formatDate(value?: string | null): string {
  if (!value) return '—'
  return new Date(value).toLocaleDateString()
}

export default function Promotions() {
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
    if (!(await confirm({ title: `Delete promotion "${p.name}"?`, message: 'This cannot be undone.', confirmLabel: 'Delete promotion', destructive: true }))) return
    setDeleteError('')
    try {
      await remove(p.id)
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Unknown error')
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
            placeholder="Search by name..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <button
            onClick={openNew}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover transition-colors font-medium"
          >
            <Plus size={13} /> Add promotion
          </button>
        </div>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{filtered.length} promotion{filtered.length === 1 ? '' : 's'}</span>
        {(error || deleteError) && <span className="text-xs text-danger ml-auto">{error || deleteError}</span>}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Promotion</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Type</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Starts</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Ends</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 w-10" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">Loading…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">No promotions yet</td></tr>
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
                <td className="px-3 py-2.5 text-ink-3">{p.type}</td>
                <td className="px-3 py-2.5 text-ink-3">{formatDate(p.starts_at)}</td>
                <td className="px-3 py-2.5 text-ink-3">{formatDate(p.ends_at)}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium capitalize', STATUS_STYLE[p.status])}>
                    {p.status}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    <button aria-label="Delete"
                      onClick={e => { e.stopPropagation(); handleDelete(p) }}
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

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={filtered.length} onPageChange={setPage} />
      {confirmDialog}
    </div>
  )
}

// ── Promotion Edit Page ──────────────────────────────────────────────────────

interface DraftLine extends PromotionLine {
  sku_code: string
  product_name: string
}

interface PromotionEditPageProps {
  token: string
  promotionId: string | null
  onCreate: (input: PromotionInput) => Promise<CreatePromotionResult>
  onUpdate: (id: string, input: PromotionInput) => Promise<Promotion>
  onClose: () => void
}

function PromotionEditPage({ token, promotionId, onCreate, onUpdate, onClose }: PromotionEditPageProps) {
  const [name, setName] = useState('')
  const [type, setType] = useState<PromotionType>('percentage')
  const [status, setStatus] = useState<PromotionStatus>('draft')
  const [startsAt, setStartsAt] = useState('')
  const [endsAt, setEndsAt] = useState('')
  const [lines, setLines] = useState<DraftLine[]>([])
  const [skuCode, setSkuCode] = useState('')
  const [resolving, setResolving] = useState(false)
  const [pickerOpen, setPickerOpen] = useState(false)
  const [loadingDetail, setLoadingDetail] = useState(promotionId !== null)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  useEffect(() => {
    if (!promotionId) return
    let cancelled = false
    setLoadingDetail(true)
    getPromotion(token, promotionId)
      .then(detail => {
        if (cancelled) return
        setName(detail.name)
        setType(detail.type)
        setStatus(detail.status)
        setStartsAt(detail.starts_at.slice(0, 16))
        setEndsAt(detail.ends_at ? detail.ends_at.slice(0, 16) : '')
        setLines((detail.lines ?? []).map(l => ({ ...l, product_name: '' })))
      })
      .catch(err => { if (!cancelled) setFormError(err instanceof Error ? err.message : 'Failed to load promotion') })
      .finally(() => { if (!cancelled) setLoadingDetail(false) })
    return () => { cancelled = true }
  }, [token, promotionId])

  async function handleAddSKU() {
    const code = skuCode.trim()
    if (!code) return
    setFormError('')
    setResolving(true)
    try {
      const result = await resolveBarcode(token, code)
      if (addLine(result.sku.id, result.sku.sku_code, result.product_name)) setSkuCode('')
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to resolve SKU')
    } finally {
      setResolving(false)
    }
  }

  // addLine appends a draft line for skuId, returning false (with a form
  // error) when the SKU is already in this promotion.
  function addLine(skuId: string, skuCode: string, productName: string): boolean {
    if (lines.some(l => l.sku_id === skuId)) {
      setFormError('That SKU is already in this promotion')
      return false
    }
    setLines(prev => [...prev, {
      id: `draft-${skuId}`, sku_id: skuId, sku_code: skuCode,
      product_name: productName, discount_type: type, discount_value: 0,
    }])
    return true
  }

  function handlePickSKU(sku: SKU, _label: string, productName: string) {
    setFormError('')
    if (addLine(sku.id, sku.sku_code, productName)) setPickerOpen(false)
  }

  function updateLineValue(skuId: string, value: string) {
    const parsed = value === '' ? 0 : parseInt(value, 10)
    setLines(prev => prev.map(l => (l.sku_id === skuId ? { ...l, discount_value: parsed } : l)))
  }

  function updateLineBatch(skuId: string, batchId: string | null) {
    setLines(prev => prev.map(l => (l.sku_id === skuId ? { ...l, batch_id: batchId } : l)))
  }

  function removeLine(skuId: string) {
    setLines(prev => prev.filter(l => l.sku_id !== skuId))
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      const input: PromotionInput = {
        name, type, starts_at: new Date(startsAt).toISOString(),
        ends_at: endsAt ? new Date(endsAt).toISOString() : null,
        status,
        lines: lines.map(l => ({
          sku_id: l.sku_id, discount_type: type, discount_value: l.discount_value,
          ...(l.batch_id ? { batch_id: l.batch_id } : {}),
        })),
      }
      if (promotionId) {
        await onUpdate(promotionId, input)
      } else {
        const result = await onCreate(input)
        if (result.warnings.length > 0) {
          window.alert(result.warnings.join('\n'))
        }
      }
      onClose()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save promotion')
    } finally {
      setSaving(false)
    }
  }

  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'

  if (loadingDetail) {
    return <div className="flex h-full items-center justify-center text-xs text-ink-4">Loading…</div>
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Promotions
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {promotionId ? name || 'Edit promotion' : 'New promotion'}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
          >
            Discard
          </button>
          <button
            type="submit" form="promotion-form" disabled={saving || !name.trim() || lines.length === 0}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : promotionId ? 'Save' : 'Create promotion'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto p-6 space-y-5">
          <form id="promotion-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border border-line p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-ink">General</h3>
                <select
                  aria-label="Status"
                  value={status}
                  onChange={e => setStatus(e.target.value as PromotionStatus)}
                  className="px-2 py-1 text-2xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
                >
                  <option value="draft">Draft</option>
                  <option value="active">Active</option>
                  <option value="ended">Ended</option>
                </select>
              </div>
              <div>
                <label htmlFor="promotion-name" className={labelClass}>Name</label>
                <input
                  id="promotion-name"
                  type="text" required autoFocus value={name} onChange={e => setName(e.target.value)}
                  className={inputClass}
                />
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label htmlFor="promotion-type" className={labelClass}>Type</label>
                  <select
                    id="promotion-type"
                    value={type} onChange={e => setType(e.target.value as PromotionType)}
                    className={inputClass}
                  >
                    <option value="percentage">Percentage</option>
                    <option value="fixed_amount">Fixed amount</option>
                  </select>
                </div>
                <div>
                  <label htmlFor="promotion-starts" className={labelClass}>Starts at</label>
                  <input
                    id="promotion-starts"
                    type="datetime-local" required value={startsAt} onChange={e => setStartsAt(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label htmlFor="promotion-ends" className={labelClass}>Ends at</label>
                  <input
                    id="promotion-ends"
                    type="datetime-local" value={endsAt} onChange={e => setEndsAt(e.target.value)}
                    className={inputClass}
                  />
                </div>
              </div>
              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>

            <div className="rounded-xl border border-line p-4 space-y-3">
              <h3 className="text-xs font-semibold text-ink">SKUs</h3>
              <div className="flex items-center gap-2">
                <input
                  type="text" placeholder="SKU code or barcode" value={skuCode} onChange={e => setSkuCode(e.target.value)}
                  className={inputClass}
                />
                <button
                  type="button" onClick={handleAddSKU} disabled={resolving || !skuCode.trim()}
                  className="shrink-0 px-3 py-2 rounded-lg bg-raised border border-line text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
                >
                  {resolving ? 'Resolving…' : 'Add SKU'}
                </button>
                <button
                  type="button" onClick={() => setPickerOpen(open => !open)}
                  className="shrink-0 inline-flex items-center gap-1 px-3 py-2 rounded-lg bg-raised border border-line text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
                >
                  <Search size={12} /> Search product
                </button>
              </div>
              {pickerOpen && <ProductSkuPicker onSelect={handlePickSKU} onClose={() => setPickerOpen(false)} />}
              {lines.length === 0 && (
                <p className="text-xs text-ink-4">No SKUs added yet.</p>
              )}
              <div className="divide-y divide-line/60">
                {lines.map(l => (
                  <div key={l.sku_id} className="flex flex-wrap sm:flex-nowrap items-center gap-3 py-2">
                    <div className="flex-1 min-w-0">
                      <p className="text-xs text-ink truncate">{l.product_name || l.sku_code}</p>
                      <p className="text-2xs font-mono text-ink-4">{l.sku_code}</p>
                      {l.finished_at && (
                        <p className="text-2xs text-ink-4">Batch finished {formatDate(l.finished_at)} — back to regular price</p>
                      )}
                    </div>
                    <div className="w-56">
                      <ClearanceBatchSelect token={token} skuId={l.sku_id} value={l.batch_id ?? null} onChange={id => updateLineBatch(l.sku_id, id)} />
                    </div>
                    <div className="w-28">
                      <label htmlFor={`discount-${l.sku_id}`} className="sr-only">Discount value</label>
                      <input
                        id={`discount-${l.sku_id}`}
                        type="number" min="0" value={l.discount_value === 0 ? '' : l.discount_value}
                        onChange={e => updateLineValue(l.sku_id, e.target.value)}
                        placeholder={type === 'percentage' ? 'bps' : 'minor units'}
                        className={inputClass}
                      />
                    </div>
                    <button aria-label="Remove"
                      type="button" onClick={() => removeLine(l.sku_id)}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors shrink-0"
                      title="Remove"
                    >
                      <Trash2 size={12} className="text-ink-3 hover:text-danger" />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          </form>

          {promotionId && <PromotionPerformanceCard token={token} promotionId={promotionId} />}
        </div>
      </div>
    </div>
  )
}

function formatMoney(minorUnits: number, currencyCode: string): string {
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || 'USD' }).format(minorUnits / 100)
  } catch {
    return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
  }
}

// PromotionPerformanceCard compares the promotion's covered SKUs' revenue/
// units during its active window (capped at "now" if still ongoing)
// against the immediately preceding period of equal length -- the Phase 0
// "Promotion Analytics" report.
function PromotionPerformanceCard({ token, promotionId }: { token: string; promotionId: string }) {
  const [report, setReport] = useState<PromotionPerformanceReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    getPromotionPerformanceReport(token, promotionId)
      .then(result => { if (!cancelled) setReport(result) })
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load performance') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, promotionId])

  const changeLabel = (percent: number | null) => {
    if (percent === null) return '— vs before'
    const up = percent >= 0
    return `${up ? '+' : ''}${percent.toFixed(1)}% vs before`
  }

  return (
    <div className="rounded-xl border border-line p-4 space-y-3">
      <div>
        <h3 className="text-xs font-semibold text-ink">Performance</h3>
        <p className="text-2xs text-ink-4 mt-0.5">
          Covered SKUs during the active window vs. the immediately preceding period of equal length
        </p>
      </div>
      {loading && <p className="text-xs text-ink-4">Loading…</p>}
      {!loading && error && <p className="text-xs text-danger">{error}</p>}
      {!loading && !error && report && (
        <div className="grid grid-cols-2 gap-3">
          <div>
            <p className="text-2xs text-ink-4 mb-1">Revenue (during)</p>
            <p className="text-sm font-semibold text-ink">{formatMoney(report.during.revenue_minor_units, report.currency_code)}</p>
            <p className={clsx('text-2xs font-medium', report.revenue_change_percent === null ? 'text-ink-4' : report.revenue_change_percent >= 0 ? 'text-success' : 'text-danger')}>
              {changeLabel(report.revenue_change_percent)}
            </p>
          </div>
          <div>
            <p className="text-2xs text-ink-4 mb-1">Units sold (during)</p>
            <p className="text-sm font-semibold text-ink">{report.during.units_sold}</p>
            <p className={clsx('text-2xs font-medium', report.units_sold_change_percent === null ? 'text-ink-4' : report.units_sold_change_percent >= 0 ? 'text-success' : 'text-danger')}>
              {changeLabel(report.units_sold_change_percent)}
            </p>
          </div>
          <div>
            <p className="text-2xs text-ink-4 mb-1">Revenue (before)</p>
            <p className="text-xs text-ink-2">{formatMoney(report.before.revenue_minor_units, report.currency_code)}</p>
          </div>
          <div>
            <p className="text-2xs text-ink-4 mb-1">Units sold (before)</p>
            <p className="text-xs text-ink-2">{report.before.units_sold}</p>
          </div>
        </div>
      )}
    </div>
  )
}
