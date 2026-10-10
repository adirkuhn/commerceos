import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import * as couponsApi from './api'
import type { Coupon, CouponInput, DiscountType, CouponCurrencyAmount } from './types'
import { apiErrorText } from '../api/apiErrorText'

const CURRENCIES = ['USD', 'EUR', 'GBP', 'CAD', 'AUD']

// CouponCurrencyAmountsEditor manages a Fixed-type coupon's manual
// per-currency amount overrides -- absent one, checkout converts
// discount_value via the tenant's configured exchange rate instead (see
// Settings > Localization). Only meaningful once the coupon exists (needs
// its id), and only for discount_type "fixed" (a Percent discount is
// already currency-agnostic).
function CouponCurrencyAmountsEditor({ couponId }: { couponId: string }) {
  const { token } = useAuth()
  const { t } = useTranslation(['coupons', 'common'])
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
      .catch(err => setError(apiErrorText(err) ?? t('overrides.loadFailed')))
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
      setError(apiErrorText(err) ?? t('overrides.saveFailed'))
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
      setError(apiErrorText(err) ?? t('overrides.deleteFailed'))
    }
  }

  return (
    <div className="rounded-xl border border-line p-4 space-y-3">
      <div>
        <h3 className="text-xs font-semibold text-ink">{t('overrides.title')}</h3>
        <p className="text-2xs text-ink-4 mt-0.5">
          {t('overrides.intro')}
        </p>
      </div>
      {loading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
      {amounts.length > 0 && (
        <div className="space-y-1.5">
          {amounts.map(a => (
            <div key={a.currency_code} className="flex items-center justify-between text-xs text-ink-2">
              <span>{t('overrides.amountOff', { currency: a.currency_code, amount: (a.amount_minor_units / 100).toFixed(2) })}</span>
              <button type="button" onClick={() => handleDelete(a.currency_code)} className="text-2xs text-ink-4 hover:text-danger transition-colors">
                {t('common:actions.delete')}
              </button>
            </div>
          ))}
        </div>
      )}
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor="coupon-currency-override-currency" className={labelClass}>{t('overrides.currency')}</label>
          <select id="coupon-currency-override-currency" value={newCurrency} onChange={e => setNewCurrency(e.target.value)} className={inputClass}>
            {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
          </select>
        </div>
        <div>
          <label htmlFor="coupon-currency-override-amount" className={labelClass}>{t('overrides.amount')}</label>
          <input
            id="coupon-currency-override-amount"
            type="number" min="0" step="0.01" placeholder={t('overrides.amountPlaceholder')} value={newAmount}
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
        {saving ? t('common:actions.saving') : t('overrides.save')}
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

export function CouponEditPage({ coupon, onCreate, onUpdate, onClose }: CouponEditPageProps) {
  const { t } = useTranslation(['coupons', 'common'])
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
      setFormError(t('edit.valueInvalid'))
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
      setFormError(apiErrorText(err) ?? t('edit.saveFailed'))
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
            <ArrowLeft size={11} /> {t('edit.back')}
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {coupon ? coupon.code : t('edit.new')}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
          >
            {t('edit.discard')}
          </button>
          <button
            type="submit" form="coupon-form" disabled={saving || !code.trim()}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? t('common:actions.saving') : coupon ? t('common:actions.save') : t('edit.create')}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto p-6 space-y-5">
          <form id="coupon-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border border-line p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-ink">{t('edit.general')}</h3>
                <label className="flex items-center gap-1.5 text-2xs text-ink-3">
                  <input type="checkbox" checked={active} onChange={e => setActive(e.target.checked)} />
                  {t('edit.active')}
                </label>
              </div>
              <div>
                <label htmlFor="coupon-code" className={labelClass}>{t('edit.code')}</label>
                <input
                  id="coupon-code"
                  type="text" required autoFocus value={code}
                  onChange={e => setCode(e.target.value.toUpperCase())}
                  className={clsx(inputClass, 'font-mono uppercase')}
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="coupon-discount-type" className={labelClass}>{t('edit.discountType')}</label>
                  <select
                    id="coupon-discount-type"
                    value={discountType} onChange={e => setDiscountType(e.target.value as DiscountType)}
                    className={inputClass}
                  >
                    <option value="percent">{t('edit.percent')}</option>
                    <option value="fixed">{t('edit.fixed')}</option>
                  </select>
                </div>
                <div>
                  <label htmlFor="coupon-discount-value" className={labelClass}>
                    {discountType === 'percent' ? t('edit.percentOff') : t('edit.amountOff')}
                  </label>
                  <input
                    id="coupon-discount-value"
                    type="number" min="0" step="0.01" required value={discountValue}
                    onChange={e => setDiscountValue(e.target.value)}
                    placeholder={discountType === 'percent' ? t('edit.percentPlaceholder') : t('edit.amountPlaceholder')}
                    className={inputClass}
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="coupon-min-subtotal" className={labelClass}>{t('edit.minSubtotal')}</label>
                  <input
                    id="coupon-min-subtotal"
                    type="number" min="0" step="0.01" value={minSubtotal}
                    onChange={e => setMinSubtotal(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label htmlFor="coupon-usage-limit" className={labelClass}>{t('edit.usageLimit')}</label>
                  <input
                    id="coupon-usage-limit"
                    type="number" min="1" step="1" value={usageLimit}
                    onChange={e => setUsageLimit(e.target.value)}
                    className={inputClass}
                  />
                </div>
              </div>
              <div>
                <label htmlFor="coupon-expires" className={labelClass}>{t('edit.expires')}</label>
                <input
                  id="coupon-expires"
                  type="datetime-local" value={expiresAt} onChange={e => setExpiresAt(e.target.value)}
                  className={inputClass}
                />
              </div>
              {coupon && (
                <p className="text-2xs text-ink-4">{t('edit.used', { count: coupon.times_used })}</p>
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
