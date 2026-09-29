import { useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import { CompanyProfileError } from './api'
import { countryOptions } from './countries'
import { useCompanyProfile } from './useCompanyProfile'
import type { CompanyProfileField, CompanyProfileInput } from './types'
import { useDateFormat } from '../../format/DateFormatProvider'

const EMPTY: CompanyProfileInput = {
  legal_name: '', trading_name: '', vat_number: '', chamber_of_commerce: '', email: '', phone: '', iban: '',
  address: { street: '', house_number: '', postal_code: '', city: '', country: '' },
}

// Settings roles that may change tenant-wide settings (matches the
// backend's RequireRoleForWrites on the settings routes).
const EDITOR_ROLES = ['customer_owner', 'customer_admin', 'super_admin']

function toInput(p: CompanyProfileInput): CompanyProfileInput {
  return {
    legal_name: p.legal_name, trading_name: p.trading_name, vat_number: p.vat_number,
    chamber_of_commerce: p.chamber_of_commerce, email: p.email, phone: p.phone, iban: p.iban,
    address: { ...p.address },
  }
}

const inputClass = 'w-full px-3 py-2 text-sm rounded-lg bg-surface border text-ink placeholder-ink-4 focus:outline-none focus:ring-2 focus:ring-focus/60 read-only:bg-sunken read-only:text-ink-3'

// Settings > Company: the tenant's business identity, printed as the seller
// block on invoices.
export function CompanyProfileSection() {
  const { t } = useTranslation(['settings', 'common'])
  const { token, user } = useAuth()
  const { formatTime } = useDateFormat()
  const { profile, loading, error: loadError, save } = useCompanyProfile(token ?? '')
  const canEdit = EDITOR_ROLES.includes(user?.role ?? '')

  // Edits only: null means "show the saved profile as is". Deriving the
  // form from the profile (instead of copying it in an effect) means no
  // render ever shows empty fields or a false "unsaved changes" state.
  const [edits, setEdits] = useState<CompanyProfileInput | null>(null)
  const [saving, setSaving] = useState(false)
  const [fieldError, setFieldError] = useState<{ field: CompanyProfileField; message: string } | null>(null)
  const [formError, setFormError] = useState('')
  const [savedAt, setSavedAt] = useState<string | null>(null)
  const fieldRefs = useRef<Partial<Record<CompanyProfileField, HTMLInputElement | HTMLSelectElement | null>>>({})

  const saved = profile ? toInput(profile) : EMPTY
  const draft = edits ?? saved
  const dirty = edits !== null && JSON.stringify(edits) !== JSON.stringify(saved)

  // Leaving the tab with unsaved edits asks first.
  useEffect(() => {
    if (!dirty) return
    const warn = (e: BeforeUnloadEvent) => { e.preventDefault() }
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  }, [dirty])

  useEffect(() => {
    if (fieldError) fieldRefs.current[fieldError.field]?.focus()
  }, [fieldError])

  function set<K extends keyof CompanyProfileInput>(key: K, value: CompanyProfileInput[K]) {
    setEdits(e => ({ ...(e ?? saved), [key]: value }))
    setSavedAt(null)
  }
  function setAddress(key: keyof CompanyProfileInput['address'], value: string) {
    setEdits(e => {
      const base = e ?? saved
      return { ...base, address: { ...base.address, [key]: value } }
    })
    setSavedAt(null)
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setSaving(true)
    setFieldError(null)
    setFormError('')
    try {
      await save(draft)
      setEdits(null)
      setSavedAt(formatTime(new Date()))
    } catch (err) {
      if (err instanceof CompanyProfileError && err.field) setFieldError({ field: err.field, message: err.message })
      else setFormError(err instanceof Error ? err.message : t('company.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  function field(
    id: CompanyProfileField | string, label: string, value: string, onChange: (v: string) => void,
    opts: { autoComplete?: string; type?: string; placeholder?: string; required?: boolean; className?: string } = {},
  ) {
    const errorHere = fieldError?.field === id
    const inputId = `company-${id.replace('.', '-')}`
    return (
      <div className={opts.className}>
        <label htmlFor={inputId} className="block text-xs font-medium text-ink-2 mb-1.5">
          {label}
        </label>
        <input
          id={inputId}
          ref={el => { fieldRefs.current[id as CompanyProfileField] = el }}
          type={opts.type ?? 'text'}
          value={value}
          onChange={e => onChange(e.target.value)}
          autoComplete={opts.autoComplete}
          placeholder={opts.placeholder}
          required={opts.required}
          readOnly={!canEdit}
          aria-invalid={errorHere || undefined}
          aria-describedby={errorHere ? `${inputId}-error` : undefined}
          className={clsx(inputClass, errorHere ? 'border-danger' : 'border-line')}
        />
        {errorHere && <p id={`${inputId}-error`} className="mt-1 text-xs text-danger">{fieldError.message}</p>}
      </div>
    )
  }

  if (loading) return <p className="text-sm text-ink-4">{t('company.loading')}</p>
  if (loadError) return <p role="alert" className="text-sm text-danger">{loadError}</p>

  const countryError = fieldError?.field === 'address.country'

  return (
    <form onSubmit={handleSubmit} className="max-w-2xl space-y-6" noValidate>
      <div>
        <h2 className="text-base font-semibold text-ink mb-1">{t('sections.company')}</h2>
        <p className="text-xs text-ink-4">{t('company.subtitle')}</p>
        {!canEdit && <p className="mt-2 text-xs text-ink-3">{t('company.readOnly')}</p>}
      </div>

      {formError && <p role="alert" className="text-sm text-danger">{formError}</p>}

      <Group title={t('company.business')}>
        {field('legal_name', t('company.legalName'), draft.legal_name, v => set('legal_name', v), { autoComplete: 'organization', required: true })}
        {field('trading_name', t('company.tradingName'), draft.trading_name, v => set('trading_name', v))}
        {field('vat_number', t('company.vatNumber'), draft.vat_number, v => set('vat_number', v), { placeholder: 'NL123456789B01' })}
        {field('chamber_of_commerce', t('company.chamberOfCommerce'), draft.chamber_of_commerce, v => set('chamber_of_commerce', v))}
      </Group>

      <Group title={t('company.contact')}>
        {field('email', t('common:fields.email'), draft.email, v => set('email', v), { type: 'email', autoComplete: 'email' })}
        {field('phone', t('company.phone'), draft.phone, v => set('phone', v), { type: 'tel', autoComplete: 'tel' })}
      </Group>

      <Group title={t('company.address')}>
        {field('address.street', t('company.street'), draft.address.street, v => setAddress('street', v), { autoComplete: 'address-line1', className: 'sm:col-span-2' })}
        {field('address.house_number', t('company.houseNumber'), draft.address.house_number, v => setAddress('house_number', v))}
        {field('address.postal_code', t('company.postalCode'), draft.address.postal_code, v => setAddress('postal_code', v), { autoComplete: 'postal-code' })}
        {field('address.city', t('company.city'), draft.address.city, v => setAddress('city', v), { autoComplete: 'address-level2' })}
        <div>
          <label htmlFor="company-address-country" className="block text-xs font-medium text-ink-2 mb-1.5">{t('company.country')}</label>
          <select
            id="company-address-country"
            ref={el => { fieldRefs.current['address.country'] = el }}
            value={draft.address.country}
            onChange={e => setAddress('country', e.target.value)}
            disabled={!canEdit}
            autoComplete="country"
            aria-invalid={countryError || undefined}
            aria-describedby={countryError ? 'company-address-country-error' : undefined}
            className={clsx(inputClass, countryError ? 'border-danger' : 'border-line', 'disabled:bg-sunken disabled:text-ink-3')}
          >
            <option value="">{t('company.notSet')}</option>
            {countryOptions(draft.address.country).map(c => <option key={c.code} value={c.code}>{c.name}</option>)}
          </select>
          {countryError && <p id="company-address-country-error" className="mt-1 text-xs text-danger">{fieldError.message}</p>}
        </div>
      </Group>

      <Group title={t('company.payment')}>
        {field('iban', 'IBAN', draft.iban, v => set('iban', v), { placeholder: 'NL91 ABNA 0417 1643 00' })}
      </Group>

      {canEdit && (
        <div className="flex items-center gap-3">
          <button
            type="submit"
            disabled={!dirty || saving}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-sm font-medium hover:bg-accent-solid-hover disabled:opacity-50"
          >
            {saving ? t('common:actions.saving') : t('common:actions.saveChanges')}
          </button>
          <p role="status" className="text-xs text-ink-3">{savedAt && t('savedAt', { time: savedAt })}</p>
        </div>
      )}
    </form>
  )
}

function Group({ title, children }: { title: string; children: ReactNode }) {
  return (
    <fieldset className="space-y-3">
      <legend className="text-xs font-semibold uppercase tracking-wider text-ink-4 mb-2">{title}</legend>
      <div className="grid gap-3 sm:grid-cols-2">{children}</div>
    </fieldset>
  )
}
