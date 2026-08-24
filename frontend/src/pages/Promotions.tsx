import { useEffect, useState } from 'react'
import { Plus, Search, Tag, Trash2, ArrowLeft } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { usePromotions } from '../promotions/hooks/usePromotions'
import { getPromotion } from '../promotions/api'
import { resolveBarcode } from '../pms/api'
import type { Promotion, PromotionInput, PromotionLine, PromotionStatus, PromotionType, CreatePromotionResult } from '../promotions/types'

const PAGE_SIZE = 20

const STATUS_STYLE: Record<PromotionStatus, string> = {
  draft: 'bg-gray-500/15 text-gray-400',
  active: 'bg-green-500/15 text-green-400',
  ended: 'bg-gray-500/15 text-gray-500',
}

function formatDate(value?: string | null): string {
  if (!value) return '—'
  return new Date(value).toLocaleDateString()
}

export default function Promotions() {
  const { token } = useAuth()
  const { promotions, loading, error, create, update, remove } = usePromotions(token ?? '')

  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [editingId, setEditingId] = useState<string | null | 'new'>(null)
  const [deleteError, setDeleteError] = useState('')

  function handleSearchChange(value: string) {
    setSearch(value)
    setPage(1)
  }

  const filtered = promotions.filter(p => p.name.toLowerCase().includes(search.toLowerCase()))
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  async function handleDelete(p: Promotion) {
    if (!window.confirm(`Delete promotion "${p.name}"? This cannot be undone.`)) return
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
        token={token ?? ''}
        promotionId={editingId === 'new' ? null : editingId}
        onCreate={create}
        onUpdate={update}
        onClose={() => setEditingId(null)}
      />
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
          <input
            value={search}
            onChange={e => handleSearchChange(e.target.value)}
            placeholder="Search by name..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white dark:border-[#2D3748] border-gray-200 border dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <button
            onClick={() => setEditingId('new')}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 transition-colors font-medium"
          >
            <Plus size={13} /> Add Promotion
          </button>
        </div>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">{filtered.length} promotion{filtered.length === 1 ? '' : 's'}</span>
        {(error || deleteError) && <span className="text-xs text-red-400 ml-auto">{error || deleteError}</span>}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Promotion</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Type</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Starts</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Ends</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 w-10" />
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={7} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No promotions yet</td></tr>
            )}
            {paged.map(p => (
              <tr
                key={p.id}
                onClick={() => setEditingId(p.id)}
                className="group hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-[#2D3748] bg-gray-200 flex items-center justify-center">
                    <Tag size={12} className="dark:text-gray-500 text-gray-400" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[280px]">
                  <p className="font-medium dark:text-gray-100 text-gray-900 truncate">{p.name}</p>
                </td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{p.type}</td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{formatDate(p.starts_at)}</td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{formatDate(p.ends_at)}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium capitalize', STATUS_STYLE[p.status])}>
                    {p.status}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={e => { e.stopPropagation(); handleDelete(p) }}
                      className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors"
                      title="Delete"
                    >
                      <Trash2 size={12} className="dark:text-gray-400 text-gray-500 hover:text-red-400" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={filtered.length} onPageChange={setPage} />
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
      if (lines.some(l => l.sku_id === result.sku.id)) {
        setFormError('That SKU is already in this promotion')
        return
      }
      setLines(prev => [...prev, {
        id: `draft-${result.sku.id}`, sku_id: result.sku.id, sku_code: result.sku.sku_code,
        product_name: result.product_name, discount_type: type, discount_value: 0,
      }])
      setSkuCode('')
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to resolve SKU')
    } finally {
      setResolving(false)
    }
  }

  function updateLineValue(skuId: string, value: string) {
    const parsed = value === '' ? 0 : parseInt(value, 10)
    setLines(prev => prev.map(l => (l.sku_id === skuId ? { ...l, discount_value: parsed } : l)))
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
        lines: lines.map(l => ({ sku_id: l.sku_id, discount_type: type, discount_value: l.discount_value })),
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

  const labelClass = 'block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500 transition-shadow'

  if (loadingDetail) {
    return <div className="flex h-full items-center justify-center text-xs dark:text-gray-500 text-gray-400">Loading…</div>
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Promotions
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">
            {promotionId ? name || 'Edit Promotion' : 'New Promotion'}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs font-semibold dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50 transition-colors"
          >
            Discard
          </button>
          <button
            type="submit" form="promotion-form" disabled={saving || !name.trim() || lines.length === 0}
            className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : promotionId ? 'Save' : 'Create Promotion'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto p-6 space-y-5">
          <form id="promotion-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">General</h3>
                <select
                  aria-label="Status"
                  value={status}
                  onChange={e => setStatus(e.target.value as PromotionStatus)}
                  className="px-2 py-1 text-[10px] rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none"
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
                <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>

            <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
              <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">SKUs</h3>
              <div className="flex items-center gap-2">
                <input
                  type="text" placeholder="SKU code" value={skuCode} onChange={e => setSkuCode(e.target.value)}
                  className={inputClass}
                />
                <button
                  type="button" onClick={handleAddSKU} disabled={resolving || !skuCode.trim()}
                  className="shrink-0 px-3 py-2 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs font-medium dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50 disabled:opacity-50 transition-colors"
                >
                  {resolving ? 'Resolving…' : 'Add SKU'}
                </button>
              </div>
              {lines.length === 0 && (
                <p className="text-xs dark:text-gray-500 text-gray-400">No SKUs added yet.</p>
              )}
              <div className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
                {lines.map(l => (
                  <div key={l.sku_id} className="flex items-center gap-3 py-2">
                    <div className="flex-1 min-w-0">
                      <p className="text-xs dark:text-gray-200 text-gray-800 truncate">{l.product_name || l.sku_code}</p>
                      <p className="text-[10px] font-mono dark:text-gray-500 text-gray-400">{l.sku_code}</p>
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
                    <button
                      type="button" onClick={() => removeLine(l.sku_id)}
                      className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors shrink-0"
                      title="Remove"
                    >
                      <Trash2 size={12} className="dark:text-gray-400 text-gray-500 hover:text-red-400" />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
