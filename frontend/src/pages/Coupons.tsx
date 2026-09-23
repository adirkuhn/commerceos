import { useState, useEffect } from 'react'
import { Plus, Search, Ticket, Trash2, ArrowLeft } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useCoupons } from '../coupons/hooks/useCoupons'
import * as couponsApi from '../coupons/api'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useQueryPage, useQueryString } from '../routing/useQueryState'
import { RecordLoading, RecordNotFound } from '../components/common/RecordStatus'
import type { Coupon, CouponInput, DiscountType, CouponCurrencyAmount } from '../coupons/types'
import { useConfirm } from '../components/common/useConfirm'

const PAGE_SIZE = 20
const CURRENCIES = ['USD', 'EUR', 'GBP', 'CAD', 'AUD']

function formatDate(value?: string | null): string {
  if (!value) return '—'
  return new Date(value).toLocaleDateString()
}

function formatDiscount(c: Coupon): string {
  return c.discount_type === 'percent' ? `${(c.discount_value / 100).toFixed(2)}%` : `${(c.discount_value / 100).toFixed(2)} off`
}

export default function Coupons() {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { coupons, loading, error, create, update, remove } = useCoupons(token ?? '')

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
    if (!(await confirm({ title: `Delete coupon "${c.code}"?`, message: 'This cannot be undone.', confirmLabel: 'Delete coupon', destructive: true }))) return
    setActionError('')
    try {
      await remove(c.id)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  if (editingId !== null) {
    const editing = editingId === NEW_RECORD ? null : coupons.find(c => c.id === editingId) ?? null
    if (editingId !== NEW_RECORD && !editing) {
      return loading ? <RecordLoading /> : <RecordNotFound label="Coupon" onBack={close} />
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
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={search}
            onChange={e => handleSearchChange(e.target.value)}
            placeholder="Search by code..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <button
            onClick={openNew}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover transition-colors font-medium"
          >
            <Plus size={13} /> Add coupon
          </button>
        </div>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{filtered.length} coupon{filtered.length === 1 ? '' : 's'}</span>
        {(error || actionError) && <span className="text-xs text-danger ml-auto">{error || actionError}</span>}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Code</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Discount</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Usage</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Expires</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 w-10" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">Loading…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">No coupons yet</td></tr>
            )}
            {paged.map(c => (
              <tr
                key={c.id}
                onClick={() => openEdit(c.id)}
                className="group hover:bg-hover/60 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-night-700 bg-gray-200 flex items-center justify-center">
                    <Ticket size={12} className="text-ink-4" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[200px]">
                  <p className="font-medium font-mono text-ink truncate">{c.code}</p>
                </td>
                <td className="px-3 py-2.5 text-ink-3">{formatDiscount(c)}</td>
                <td className="px-3 py-2.5 text-ink-3">
                  {c.times_used}{c.usage_limit != null ? ` / ${c.usage_limit}` : ''}
                </td>
                <td className="px-3 py-2.5 text-ink-3">{formatDate(c.expires_at)}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium', c.active ? 'bg-green-500/15 text-success' : 'bg-gray-500/15 text-ink-4')}>
                    {c.active ? 'Active' : 'Inactive'}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    <button aria-label="Delete"
                      onClick={e => { e.stopPropagation(); handleDelete(c) }}
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

// CouponCurrencyAmountsEditor manages a Fixed-type coupon's manual
// per-currency amount overrides -- absent one, checkout converts
// discount_value via the tenant's configured exchange rate instead (see
// Settings > Localization). Only meaningful once the coupon exists (needs
// its id), and only for discount_type "fixed" (a Percent discount is
// already currency-agnostic).
function CouponCurrencyAmountsEditor({ couponId }: { couponId: string }) {
  const { token } = useAuth()
  const [amounts, setAmounts] = useState<CouponCurrencyAmount[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [newCurrency, setNewCurrency] = useState(CURRENCIES[0])
  const [newAmount, setNewAmount] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!token) return
    setLoading(true)
    couponsApi.listCouponCurrencyAmounts(token, couponId)
      .then(setAmounts)
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to load currency overrides'))
      .finally(() => setLoading(false))
  }, [token, couponId])

  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'

  async function handleAdd() {
    if (!token || !newAmount) return
    setSaving(true)
    setError('')
    try {
      const result = await couponsApi.upsertCouponCurrencyAmount(token, couponId, {
        currency_code: newCurrency, amount_minor_units: Math.round(parseFloat(newAmount) * 100),
      })
      setAmounts(prev => (prev.some(a => a.currency_code === result.currency_code) ? prev.map(a => (a.currency_code === result.currency_code ? result : a)) : [...prev, result]))
      setNewAmount('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save currency override')
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(currencyCode: string) {
    if (!token) return
    setError('')
    try {
      await couponsApi.deleteCouponCurrencyAmount(token, couponId, currencyCode)
      setAmounts(prev => prev.filter(a => a.currency_code !== currencyCode))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete currency override')
    }
  }

  return (
    <div className="rounded-xl border border-line p-4 space-y-3">
      <div>
        <h3 className="text-xs font-semibold text-ink">Currency overrides</h3>
        <p className="text-2xs text-ink-4 mt-0.5">
          Set an exact amount off for a channel running in another currency, instead of the auto-converted rate.
        </p>
      </div>
      {loading && <p className="text-xs text-ink-4">Loading…</p>}
      {amounts.length > 0 && (
        <div className="space-y-1.5">
          {amounts.map(a => (
            <div key={a.currency_code} className="flex items-center justify-between text-xs text-ink-2">
              <span>{a.currency_code}: {(a.amount_minor_units / 100).toFixed(2)} off</span>
              <button type="button" onClick={() => handleDelete(a.currency_code)} className="text-2xs text-ink-4 hover:text-danger transition-colors">
                Delete
              </button>
            </div>
          ))}
        </div>
      )}
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor="coupon-currency-override-currency" className={labelClass}>Currency</label>
          <select id="coupon-currency-override-currency" value={newCurrency} onChange={e => setNewCurrency(e.target.value)} className={inputClass}>
            {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
          </select>
        </div>
        <div>
          <label htmlFor="coupon-currency-override-amount" className={labelClass}>Override amount</label>
          <input
            id="coupon-currency-override-amount"
            type="number" min="0" step="0.01" placeholder="e.g. 5.00" value={newAmount}
            onChange={e => setNewAmount(e.target.value)}
            className={inputClass}
          />
        </div>
      </div>
      <button
        type="button"
        onClick={handleAdd}
        disabled={saving || !newAmount}
        className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
      >
        {saving ? 'Saving…' : 'Add / update'}
      </button>
      {error && <p className="text-xs text-danger">{error}</p>}
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

  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Coupons
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {coupon ? coupon.code : 'New coupon'}
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
            type="submit" form="coupon-form" disabled={saving || !code.trim()}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : coupon ? 'Save' : 'Create coupon'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto p-6 space-y-5">
          <form id="coupon-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border border-line p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-ink">General</h3>
                <label className="flex items-center gap-1.5 text-2xs text-ink-3">
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
                <p className="text-2xs text-ink-4">Used {coupon.times_used} time{coupon.times_used === 1 ? '' : 's'}</p>
              )}
              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>
          </form>
          {coupon && discountType === 'fixed' && <CouponCurrencyAmountsEditor couponId={coupon.id} />}
        </div>
      </div>
    </div>
  )
}
