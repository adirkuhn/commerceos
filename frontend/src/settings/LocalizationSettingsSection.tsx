import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useTenantSettings } from './hooks/useTenantSettings'
import { useApplyDateFormatSettings } from '../format/DateFormatProvider'
import { CURRENCY_CODES, currencyOptions } from './currencyOptions'
import { useExchangeRates } from '../fx/hooks/useExchangeRates'
import { useConfirm } from '../components/common/useConfirm'
import { applyCompanyLanguage } from '../i18n/i18n'
import { DEFAULT_LOCALE, LOCALE_NAMES, LOCALES } from '../i18n/locale'
import { apiErrorText } from '../api/apiErrorText'

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

// Named in each language itself, like any language picker.
const LANGUAGE_OPTIONS = LOCALES.map(l => ({ value: l, label: LOCALE_NAMES[l] }))

export function LocalizationSettingsSection() {
  const { token } = useAuth()
  const { settings, loading, error, save } = useTenantSettings(token ?? '')
  const applyDateFormat = useApplyDateFormatSettings()
  const { t } = useTranslation(['settings', 'common'])

  const [defaultCurrency, setDefaultCurrency] = useState<string>(CURRENCY_CODES[0])
  const [timezone, setTimezone] = useState(TIMEZONE_OPTIONS[0].value)
  const [dateFormat, setDateFormat] = useState(DATE_FORMAT_OPTIONS[0].value)
  const [language, setLanguage] = useState<string>(DEFAULT_LOCALE)
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
      const next = await save({ default_currency: defaultCurrency, timezone, date_format: dateFormat, language })
      applyDateFormat(next)
      await applyCompanyLanguage(next.language)
      setSaved(true)
    } catch (err) {
      setSaveError(apiErrorText(err) ?? t('saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  const fields: { id: string; label: string; value: string; onChange: (v: string) => void; options: { value: string; label: string }[]; help?: string }[] = [
    { id: 'localization-currency', label: t('localization.defaultCurrency'), value: defaultCurrency, onChange: setDefaultCurrency, options: currencyOptions() },
    { id: 'localization-timezone', label: t('localization.timezone'), value: timezone, onChange: setTimezone, options: TIMEZONE_OPTIONS },
    { id: 'localization-date-format', label: t('localization.dateFormat'), value: dateFormat, onChange: setDateFormat, options: DATE_FORMAT_OPTIONS },
    { id: 'localization-language', label: t('localization.language'), value: language, onChange: setLanguage, options: LANGUAGE_OPTIONS, help: t('localization.languageHelp') },
  ]

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-base font-semibold text-ink mb-1">{t('localization.title')}</h2>
        <p className="text-xs text-ink-4">{t('localization.subtitle')}</p>
      </div>
      {loading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
      <div className="space-y-4">
        {fields.map(({ id, label, value, onChange, options, help }) => (
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
            {help && <p className="mt-1 text-2xs text-ink-4">{help}</p>}
          </div>
        ))}
      </div>
      {(error || saveError) && <p className="text-xs text-danger">{error || saveError}</p>}
      {saved && <p className="text-xs text-success">{t('savedMessage')}</p>}
      <button
        onClick={handleSave}
        disabled={saving}
        className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50"
      >
        {saving ? t('common:actions.saving') : t('common:actions.saveChanges')}
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
  const { t } = useTranslation(['settings', 'common'])
  const { token } = useAuth()
  const { settings } = useTenantSettings(token ?? '')
  const defaultCurrencyLabel = settings?.default_currency || t('fx.yourDefaultCurrency')
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
      setFormError(t('fx.ratePositive'))
      return
    }
    setSaving(true)
    try {
      await upsert({ currency_code: currencyCode.toUpperCase(), rate_basis_points: Math.round(parsed * 10000) })
      setCurrencyCode('')
      setRateValue('')
    } catch (err) {
      setFormError(apiErrorText(err) ?? t('fx.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(code: string) {
    if (!(await confirm({ title: t('fx.deleteTitle', { code }), message: t('fx.deleteMessage'), confirmLabel: t('fx.deleteConfirm'), destructive: true }))) return
    setFormError('')
    try {
      await remove(code)
    } catch (err) {
      setFormError(apiErrorText(err) ?? t('fx.deleteFailed'))
    }
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <div>
        <h3 className="text-sm font-semibold text-ink">{t('fx.title')}</h3>
        <p className="text-xs text-ink-4 mt-0.5">
          {t('fx.help')}
        </p>
      </div>

      {error && <p className="text-xs text-danger">{error}</p>}
      {loading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}

      <div className="space-y-1.5">
        {rates.map(r => (
          <div key={r.currency_code} className="flex items-center justify-between text-xs text-ink-2 py-1">
            <span>1 {defaultCurrencyLabel} = <span className="font-medium">{(r.rate_basis_points / 10000).toFixed(4)}</span> {r.currency_code}</span>
            <button
              onClick={() => handleDelete(r.currency_code)}
              className="text-2xs text-ink-4 hover:text-danger transition-colors"
            >
              {t('common:actions.delete')}
            </button>
          </div>
        ))}
        {!loading && rates.length === 0 && <p className="text-xs text-ink-4">{t('fx.none')}</p>}
      </div>

      <form onSubmit={handleAdd} className="flex items-end gap-2 pt-2 border-t border-line">
        <div className="w-20">
          <label htmlFor="fx-currency" className="block text-2xs text-ink-4 caps-label mb-1.5">{t('common:fields.currency')}</label>
          <input
            id="fx-currency"
            value={currencyCode}
            maxLength={3}
            placeholder={'USD'}
            onChange={e => setCurrencyCode(e.target.value.toUpperCase())}
            className="w-full px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div className="w-28">
          <label htmlFor="fx-rate" className="block text-2xs text-ink-4 caps-label mb-1.5">{t('fx.rate')}</label>
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
          {saving ? t('common:actions.saving') : t('fx.addUpdate')}
        </button>
      </form>
      {formError && <p className="text-xs text-danger">{formError}</p>}
      {confirmDialog}
    </div>
  )
}
