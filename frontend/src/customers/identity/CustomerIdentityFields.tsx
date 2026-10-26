import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import type { CustomerType } from '../types'
import type { IdentityDraft } from './customerIdentity'

const TYPES: CustomerType[] = ['company', 'private']

interface CustomerIdentityFieldsProps {
  idPrefix: string
  value: IdentityDraft
  onChange: (value: IdentityDraft) => void
  // On create: the person named for a company becomes its contact person.
  showContactPerson: boolean
  showPhone?: boolean
}

// Who the customer is: Company (company name, KVK, VAT ID, contact person)
// or Private (first and last name), plus email and client number. Shared by
// the customer form and the order's new-customer dialog.
export function CustomerIdentityFields({ idPrefix, value, onChange, showContactPerson, showPhone }: CustomerIdentityFieldsProps) {
  const { t } = useTranslation('customers')
  const set = (field: keyof IdentityDraft) => (e: React.ChangeEvent<HTMLInputElement>) => onChange({ ...value, [field]: e.target.value })
  const label = 'block text-2xs font-medium text-ink-4 caps-label mb-1.5'
  const input = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'
  const field = (name: keyof IdentityDraft, text: string, opts: { type?: string; placeholder?: string; span?: boolean; autoFocus?: boolean } = {}) => (
    <div className={clsx(opts.span && 'col-span-2')}>
      <label htmlFor={`${idPrefix}-${name}`} className={label}>{text}</label>
      <input id={`${idPrefix}-${name}`} type={opts.type ?? 'text'} value={value[name]} onChange={set(name)}
        placeholder={opts.placeholder} autoFocus={opts.autoFocus} className={input} />
    </div>
  )
  const company = value.type === 'company'

  return (
    <div className="space-y-3">
      <fieldset>
        <legend className={label}>{t('identity.type')}</legend>
        <div role="radiogroup" className="grid grid-cols-2 gap-1 p-1 rounded-lg bg-fill max-w-xs">
          {TYPES.map(type => (
            <button key={type} type="button" role="radio" aria-checked={value.type === type}
              onClick={() => onChange({ ...value, type })}
              className={clsx('px-2 py-1.5 rounded-md text-xs font-medium transition-colors',
                value.type === type ? 'bg-raised text-ink shadow-sm' : 'text-ink-3 hover:text-ink')}>
              {t(`identity.types.${type}`)}
            </button>
          ))}
        </div>
      </fieldset>

      <div className="grid grid-cols-2 gap-3">
        {company ? (
          <>
            {field('company_name', t('identity.companyName'), { span: true, autoFocus: true })}
            {field('kvk_number', t('identity.kvkNumber'))}
            {field('vat_number', t('identity.vatNumber'))}
          </>
        ) : (
          <>
            {field('first_name', t('edit.firstName'), { autoFocus: true })}
            {field('last_name', t('edit.lastName'))}
          </>
        )}
        {field('email', t('identity.email'), { type: 'email' })}
        {showPhone ? field('phone', t('identity.phone'), { type: 'tel' }) : field('client_number', t('identity.clientNumber'), { placeholder: t('identity.automatic') })}
        {showPhone && field('client_number', t('identity.clientNumber'), { placeholder: t('identity.automatic') })}
      </div>

      {company && showContactPerson && (
        <fieldset className="space-y-2">
          <legend className="text-xs font-semibold text-ink">{t('identity.contactPerson')}</legend>
          <div className="grid grid-cols-2 gap-3">
            {field('first_name', t('identity.contactFirstName'))}
            {field('last_name', t('identity.contactLastName'))}
          </div>
        </fieldset>
      )}
    </div>
  )
}
