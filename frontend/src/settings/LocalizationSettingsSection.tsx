import { useState } from 'react'
import { useAuth } from '../auth/AuthContext'
import { useTenantSettings } from './hooks/useTenantSettings'
import { useApplyDateFormatSettings } from '../format/DateFormatProvider'
import { CURRENCY_OPTIONS } from './currencyOptions'
import { useExchangeRates } from '../fx/hooks/useExchangeRates'
import { toInput } from './tenantSettingsInput'
import { useConfirm } from '../components/common/useConfirm'

// ── Localization settings section ─────────────────────────────────────────────

const TIMEZONE_OPTIONS = [
  { value: 'Europe/Amsterdam', label: 'Europe/Amsterdam (UTC+2)' },
  { value: 'Europe/London', label: 'Europe/London (UTC+1)' },
  { value: 'Europe/Paris', label: 'Europe/Paris (UTC+2)' },
  { value: 'America/New_York', label: 'America/New_York (UTC-4)' },
]

const DATE_FORMAT_OPTIONS = [
  { value: 'DD/MM/YYYY', label: 'DD/MM/YYYY' },
  { value: 'MM/DD/YYYY', label: 'MM/DD/YYYY' },
  { value: 'YYYY-MM-DD', label: 'YYYY-MM-DD' },
]

const LANGUAGE_OPTIONS = [
  { value: 'en', label: 'English (EN)' },
  { value: 'nl', label: 'Dutch (NL)' },
  { value: 'de', label: 'German (DE)' },
  { value: 'fr', label: 'French (FR)' },
]

export function LocalizationSettingsSection() {
  const { token } = useAuth()
  const { settings, loading, error, save } = useTenantSettings(token ?? '')
  const applyDateFormat = useApplyDateFormatSettings()

  const [defaultCurrency, setDefaultCurrency] = useState(CURRENCY_OPTIONS[0].value)
  const [timezone, setTimezone] = useState(TIMEZONE_OPTIONS[0].value)
  const [dateFormat, setDateFormat] = useState(DATE_FORMAT_OPTIONS[0].value)
  const [language, setLanguage] = useState(LANGUAGE_OPTIONS[0].value)
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [saved, setSaved] = useState(false)

  if (!hydrated && settings) {
    if (settings.default_currency) setDefaultCurrency(settings.default_currency)
    if (settings.timezone) setTimezone(settings.timezone)
    if (settings.date_format) setDateFormat(settings.date_format)
    if (settings.language) setLanguage(settings.language)
    setHydrated(true)
  }

  async function handleSave() {
    setSaving(true)
    setSaveError('')
    setSaved(false)
    try {
      applyDateFormat(await save({ ...toInput(settings), default_currency: defaultCurrency, timezone, date_format: dateFormat, language }))
      setSaved(true)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save settings')
    } finally {
      setSaving(false)
    }
  }

  const fields: { id: string; label: string; value: string; onChange: (v: string) => void; options: { value: string; label: string }[] }[] = [
    { id: 'localization-currency', label: 'Default currency', value: defaultCurrency, onChange: setDefaultCurrency, options: CURRENCY_OPTIONS },
    { id: 'localization-timezone', label: 'Timezone', value: timezone, onChange: setTimezone, options: TIMEZONE_OPTIONS },
    { id: 'localization-date-format', label: 'Date format', value: dateFormat, onChange: setDateFormat, options: DATE_FORMAT_OPTIONS },
    { id: 'localization-language', label: 'Language', value: language, onChange: setLanguage, options: LANGUAGE_OPTIONS },
  ]

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-base font-semibold text-ink mb-1">Localization</h2>
        <p className="text-xs text-ink-4">Currency, timezone, date formats</p>
      </div>
      {loading && <p className="text-xs text-ink-4">Loading…</p>}
      <div className="space-y-4">
        {fields.map(({ id, label, value, onChange, options }) => (
          <div key={id}>
            <label htmlFor={id} className="block text-xs font-medium text-ink-2 mb-1.5">{label}</label>
            <select
              id={id}
              value={value}
              onChange={e => { onChange(e.target.value); setSaved(false) }}
              className="w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus"
            >
              {options.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
            </select>
          </div>
        ))}
      </div>
      {(error || saveError) && <p className="text-xs text-danger">{error || saveError}</p>}
      {saved && <p className="text-xs text-success">Saved.</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50"
      >
        {saving ? 'Saving…' : 'Save changes'}
      </button>

      <ExchangeRatesCard />
    </div>
  )
}

// ExchangeRatesCard manages the tenant's fx conversion rates -- the
// fallback a SalesChannel running in a non-default currency uses to price
// a SKU/coupon with no explicit per-currency override (see domain/fx). A
// rate converts 1 minor-unit of the tenant's default currency into that
// many minor-units of the target currency.
function ExchangeRatesCard() {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { settings } = useTenantSettings(token ?? '')
  const defaultCurrencyLabel = settings?.default_currency || 'your default currency'
  const { rates, loading, error, upsert, remove } = useExchangeRates(token ?? '')
  const [currencyCode, setCurrencyCode] = useState('')
  const [rateValue, setRateValue] = useState('')
  const [formError, setFormError] = useState('')
  const [saving, setSaving] = useState(false)

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    const parsed = parseFloat(rateValue)
    if (!parsed || parsed <= 0) {
      setFormError('Rate must be a positive number')
      return
    }
    setSaving(true)
    try {
      await upsert({ currency_code: currencyCode.toUpperCase(), rate_basis_points: Math.round(parsed * 10000) })
      setCurrencyCode('')
      setRateValue('')
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save exchange rate')
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(code: string) {
    if (!(await confirm({ title: `Delete the exchange rate for ${code}?`, message: 'Channels priced in it by conversion (not a manual override) stop pricing until a new rate is set.', confirmLabel: 'Delete rate', destructive: true }))) return
    setFormError('')
    try {
      await remove(code)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to delete exchange rate')
    }
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <div>
        <h3 className="text-sm font-semibold text-ink">Exchange rates</h3>
        <p className="text-xs text-ink-4 mt-0.5">
          How much 1 unit of your default currency is worth in another currency -- used to price a sales channel
          running in that currency, unless a product/coupon has a manual override.
        </p>
      </div>

      {error && <p className="text-xs text-danger">{error}</p>}
      {loading && <p className="text-xs text-ink-4">Loading…</p>}

      <div className="space-y-1.5">
        {rates.map(r => (
          <div key={r.currency_code} className="flex items-center justify-between text-xs text-ink-2 py-1">
            <span>1 {defaultCurrencyLabel} = <span className="font-medium">{(r.rate_basis_points / 10000).toFixed(4)}</span> {r.currency_code}</span>
            <button
              onClick={() => handleDelete(r.currency_code)}
              className="text-2xs text-ink-4 hover:text-danger transition-colors"
            >
              Delete
            </button>
          </div>
        ))}
        {!loading && rates.length === 0 && <p className="text-xs text-ink-4">No exchange rates configured yet.</p>}
      </div>

      <form onSubmit={handleAdd} className="flex items-end gap-2 pt-2 border-t border-line">
        <div className="w-20">
          <label htmlFor="fx-currency" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">Currency</label>
          <input
            id="fx-currency"
            value={currencyCode}
            maxLength={3}
            placeholder="USD"
            onChange={e => setCurrencyCode(e.target.value.toUpperCase())}
            className="w-full px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div className="w-28">
          <label htmlFor="fx-rate" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">Rate</label>
          <input
            id="fx-rate"
            value={rateValue}
            placeholder="1.0876"
            onChange={e => setRateValue(e.target.value)}
            className="w-full px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <button
          type="submit"
          disabled={saving || currencyCode.length !== 3 || !rateValue}
          className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
        >
          {saving ? 'Saving…' : 'Add / update'}
        </button>
      </form>
      {formError && <p className="text-xs text-danger">{formError}</p>}
      {confirmDialog}
    </div>
  )
}
