import { useState } from 'react'
import { Plus, Search, Ticket, Trash2, ArrowLeft } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useCoupons } from '../coupons/hooks/useCoupons'
import type { Coupon, CouponInput, DiscountType } from '../coupons/types'

const PAGE_SIZE = 20

function formatDate(value?: string | null): string {
  if (!value) return '—'
  return new Date(value).toLocaleDateString()
}

function formatDiscount(c: Coupon): string {
  return c.discount_type === 'percent' ? `${(c.discount_value / 100).toFixed(2)}%` : `${(c.discount_value / 100).toFixed(2)} off`
}

export default function Coupons() {
  const { token } = useAuth()
  const { coupons, loading, error, create, update, remove } = useCoupons(token ?? '')

  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [editingId, setEditingId] = useState<string | null | 'new'>(null)
  const [actionError, setActionError] = useState('')

  function handleSearchChange(value: string) {
    setSearch(value)
    setPage(1)
  }

  const filtered = coupons.filter(c => c.code.toLowerCase().includes(search.toLowerCase()))
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  async function handleDelete(c: Coupon) {
    if (!window.confirm(`Delete coupon "${c.code}"? This cannot be undone.`)) return
    setActionError('')
    try {
      await remove(c.id)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  if (editingId !== null) {
    const editing = editingId === 'new' ? null : coupons.find(c => c.id === editingId) ?? null
    return (
      <CouponEditPage
        coupon={editing}
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
            placeholder="Search by code..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white dark:border-[#2D3748] border-gray-200 border dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <button
            onClick={() => setEditingId('new')}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 transition-colors font-medium"
          >
            <Plus size={13} /> Add Coupon
          </button>
        </div>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">{filtered.length} coupon{filtered.length === 1 ? '' : 's'}</span>
        {(error || actionError) && <span className="text-xs text-red-400 ml-auto">{error || actionError}</span>}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Code</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Discount</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Usage</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Expires</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 w-10" />
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={7} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No coupons yet</td></tr>
            )}
            {paged.map(c => (
              <tr
                key={c.id}
                onClick={() => setEditingId(c.id)}
                className="group hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-[#2D3748] bg-gray-200 flex items-center justify-center">
                    <Ticket size={12} className="dark:text-gray-500 text-gray-400" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[200px]">
                  <p className="font-medium font-mono dark:text-gray-100 text-gray-900 truncate">{c.code}</p>
                </td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{formatDiscount(c)}</td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">
                  {c.times_used}{c.usage_limit != null ? ` / ${c.usage_limit}` : ''}
                </td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{formatDate(c.expires_at)}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium', c.active ? 'bg-green-500/15 text-green-400' : 'bg-gray-500/15 text-gray-400')}>
                    {c.active ? 'Active' : 'Inactive'}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={e => { e.stopPropagation(); handleDelete(c) }}
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

// ── Coupon Edit Page ─────────────────────────────────────────────────────────

interface CouponEditPageProps {
  coupon: Coupon | null
  onCreate: (input: CouponInput) => Promise<Coupon>
  onUpdate: (id: string, input: CouponInput) => Promise<Coupon>
  onClose: () => void
}

function CouponEditPage({ coupon, onCreate, onUpdate, onClose }: CouponEditPageProps) {
  const [code, setCode] = useState(coupon?.code ?? '')
  const [discountType, setDiscountType] = useState<DiscountType>(coupon?.discount_type ?? 'percent')
  // Displayed in the units a human types -- percent (e.g. "10" for 10%) or
  // major currency units for fixed -- converted to basis-points/minor-units
  // only at submit time, same convention as SalesChannels' discount field.
  const [discountValue, setDiscountValue] = useState(coupon ? (coupon.discount_type === 'percent' ? coupon.discount_value / 100 : coupon.discount_value / 100).toString() : '')
  const [minSubtotal, setMinSubtotal] = useState(coupon?.min_subtotal_minor_units != null ? (coupon.min_subtotal_minor_units / 100).toString() : '')
  const [usageLimit, setUsageLimit] = useState(coupon?.usage_limit != null ? coupon.usage_limit.toString() : '')
  const [expiresAt, setExpiresAt] = useState(coupon?.expires_at ? coupon.expires_at.slice(0, 16) : '')
  const [active, setActive] = useState(coupon?.active ?? true)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    const parsedValue = parseFloat(discountValue)
    if (isNaN(parsedValue) || parsedValue <= 0) {
      setFormError('Discount value must be a positive number')
      return
    }
    setSaving(true)
    try {
      const input: CouponInput = {
        code: code.trim(),
        discount_type: discountType,
        discount_value: Math.round(parsedValue * 100),
        min_subtotal_minor_units: minSubtotal.trim() ? Math.round(parseFloat(minSubtotal) * 100) : null,
        usage_limit: usageLimit.trim() ? parseInt(usageLimit, 10) : null,
        expires_at: expiresAt ? new Date(expiresAt).toISOString() : null,
        active,
      }
      if (coupon) {
        await onUpdate(coupon.id, input)
      } else {
        await onCreate(input)
      }
      onClose()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save coupon')
    } finally {
      setSaving(false)
    }
  }

  const labelClass = 'block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500 transition-shadow'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Coupons
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">
            {coupon ? coupon.code : 'New Coupon'}
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
            type="submit" form="coupon-form" disabled={saving || !code.trim()}
            className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : coupon ? 'Save' : 'Create Coupon'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto p-6 space-y-5">
          <form id="coupon-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">General</h3>
                <label className="flex items-center gap-1.5 text-[11px] dark:text-gray-400 text-gray-500">
                  <input type="checkbox" checked={active} onChange={e => setActive(e.target.checked)} />
                  Active
                </label>
              </div>
              <div>
                <label htmlFor="coupon-code" className={labelClass}>Code</label>
                <input
                  id="coupon-code"
                  type="text" required autoFocus value={code}
                  onChange={e => setCode(e.target.value.toUpperCase())}
                  className={clsx(inputClass, 'font-mono uppercase')}
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="coupon-discount-type" className={labelClass}>Discount type</label>
                  <select
                    id="coupon-discount-type"
                    value={discountType} onChange={e => setDiscountType(e.target.value as DiscountType)}
                    className={inputClass}
                  >
                    <option value="percent">Percent</option>
                    <option value="fixed">Fixed amount</option>
                  </select>
                </div>
                <div>
                  <label htmlFor="coupon-discount-value" className={labelClass}>
                    {discountType === 'percent' ? 'Percent off' : 'Amount off'}
                  </label>
                  <input
                    id="coupon-discount-value"
                    type="number" min="0" step="0.01" required value={discountValue}
                    onChange={e => setDiscountValue(e.target.value)}
                    placeholder={discountType === 'percent' ? 'e.g. 10' : 'e.g. 5.00'}
                    className={inputClass}
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="coupon-min-subtotal" className={labelClass}>Min. subtotal (optional)</label>
                  <input
                    id="coupon-min-subtotal"
                    type="number" min="0" step="0.01" value={minSubtotal}
                    onChange={e => setMinSubtotal(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label htmlFor="coupon-usage-limit" className={labelClass}>Usage limit (optional)</label>
                  <input
                    id="coupon-usage-limit"
                    type="number" min="1" step="1" value={usageLimit}
                    onChange={e => setUsageLimit(e.target.value)}
                    className={inputClass}
                  />
                </div>
              </div>
              <div>
                <label htmlFor="coupon-expires" className={labelClass}>Expires at (optional)</label>
                <input
                  id="coupon-expires"
                  type="datetime-local" value={expiresAt} onChange={e => setExpiresAt(e.target.value)}
                  className={inputClass}
                />
              </div>
              {coupon && (
                <p className="text-[11px] dark:text-gray-500 text-gray-400">Used {coupon.times_used} time{coupon.times_used === 1 ? '' : 's'}</p>
              )}
              {formError && (
                <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
