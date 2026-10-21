import { useState, type FormEvent, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import type { ShippingAddress, ShippingAddresses, UpdateShippingAddressesInput } from '../types'
import { ShippingAddressFields } from './ShippingAddressFields'
import {
  addressOfDraft, draftOfAddress, validateAddressDraft,
  type AddressDraft, type AddressErrors, type AddressField,
} from './shippingAddressDraft'
import { COMPANY_PROFILE_PATH } from '../../settings/settingsPaths'
import { apiErrorText } from '../../api/apiErrorText'

interface ShippingAddressesFormProps {
  initial: ShippingAddresses
  canEdit: boolean
  // Rejects on failure so the form can show why.
  onSave: (input: UpdateShippingAddressesInput) => Promise<void>
}

// One address being edited: whether it is used at all (null = the
// fallback: company address / same as sender), its fields and errors.
interface AddressChoice {
  custom: boolean
  draft: AddressDraft
  errors: AddressErrors
}

function choiceOf(address: ShippingAddress | null): AddressChoice {
  return { custom: address !== null, draft: draftOfAddress(address), errors: {} }
}

// The sender and return address choices; saves both together (the PUT
// replaces both).
export function ShippingAddressesForm({ initial, canEdit, onSave }: ShippingAddressesFormProps) {
  const { t } = useTranslation(['shipping', 'common'])
  const [sender, setSender] = useState(() => choiceOf(initial.sender_address))
  const [returns, setReturns] = useState(() => choiceOf(initial.return_address))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  const edit = (set: typeof setSender) => (field: AddressField, value: string) => {
    setSaved(false)
    set(c => ({ ...c, draft: { ...c.draft, [field]: value }, errors: { ...c.errors, [field]: undefined } }))
  }
  const choose = (set: typeof setSender, custom: boolean) => { setSaved(false); set(c => ({ ...c, custom })) }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const senderErrors = sender.custom ? validateAddressDraft(sender.draft) : {}
    const returnErrors = returns.custom ? validateAddressDraft(returns.draft) : {}
    setSender(c => ({ ...c, errors: senderErrors }))
    setReturns(c => ({ ...c, errors: returnErrors }))
    if (Object.keys(senderErrors).length > 0 || Object.keys(returnErrors).length > 0) return
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      await onSave({
        sender_address: sender.custom ? addressOfDraft(sender.draft) : null,
        return_address: returns.custom ? addressOfDraft(returns.draft) : null,
      })
      setSaved(true)
    } catch (err) {
      setError(apiErrorText(err) ?? t('addresses.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} noValidate className="space-y-3">
      <AddressGroup
        legend={t('addresses.sender')} name="sender-mode" choice={sender} disabled={!canEdit}
        fallbackLabel={t('addresses.senderCompany')} customLabel={t('addresses.senderCustom')}
        onChoose={custom => choose(setSender, custom)} onChange={edit(setSender)}
        fallback={<CompanyAddress address={initial.company_address} />}
      />
      <AddressGroup
        legend={t('addresses.return')} hint={t('addresses.returnHint')} name="return-mode" choice={returns} disabled={!canEdit}
        fallbackLabel={t('addresses.returnSame')} customLabel={t('addresses.returnCustom')}
        onChoose={custom => choose(setReturns, custom)} onChange={edit(setReturns)}
      />
      {canEdit ? (
        <button
          type="submit" disabled={saving}
          className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
        >
          {saving ? t('common:actions.saving') : t('addresses.save')}
        </button>
      ) : (
        <p className="text-2xs text-ink-4">{t('addresses.readOnly')}</p>
      )}
      {error && <p role="alert" className="text-xs text-danger">{error}</p>}
      {saved && <p role="status" className="text-xs text-success">{t('addresses.saved')}</p>}
    </form>
  )
}

interface AddressGroupProps {
  legend: string
  hint?: string
  // The radio group's name, also the field id prefix.
  name: string
  choice: AddressChoice
  disabled: boolean
  fallbackLabel: string
  customLabel: string
  // Shown under the fallback option while it is picked.
  fallback?: ReactNode
  onChoose: (custom: boolean) => void
  onChange: (field: AddressField, value: string) => void
}

function AddressGroup({ legend, hint, name, choice, disabled, fallbackLabel, customLabel, fallback, onChoose, onChange }: AddressGroupProps) {
  return (
    <fieldset className="rounded-lg border border-line p-3">
      <legend className="px-1 text-xs font-semibold text-ink-2">{legend}</legend>
      {hint && <p className="text-2xs text-ink-4 mb-1">{hint}</p>}
      <div className="flex flex-wrap gap-x-4 gap-y-1">
        <label className="flex items-center gap-1.5 text-xs text-ink-2">
          <input type="radio" name={name} checked={!choice.custom} disabled={disabled} onChange={() => onChoose(false)} />
          {fallbackLabel}
        </label>
        <label className="flex items-center gap-1.5 text-xs text-ink-2">
          <input type="radio" name={name} checked={choice.custom} disabled={disabled} onChange={() => onChoose(true)} />
          {customLabel}
        </label>
      </div>
      {choice.custom
        ? <ShippingAddressFields idPrefix={name} draft={choice.draft} errors={choice.errors} disabled={disabled} onChange={onChange} />
        : fallback}
    </fieldset>
  )
}

// The company profile address the sender falls back to, read-only.
function CompanyAddress({ address }: { address: ShippingAddress | null }) {
  const { t } = useTranslation('shipping')
  if (!address) {
    return (
      <p className="mt-2 text-2xs text-warning">
        {t('addresses.noCompanyAddress')}{' '}
        <Link to={COMPANY_PROFILE_PATH} className="text-accent hover:underline">{t('addresses.companyProfileLink')}</Link>
      </p>
    )
  }
  return (
    <address className="mt-2 not-italic text-2xs text-ink-3 leading-relaxed">
      {address.company}<br />
      {`${address.street} ${address.number}${address.number_suffix ? ` ${address.number_suffix}` : ''}`}<br />
      {`${address.postal_code} ${address.city}`}<br />
      {address.cc}
    </address>
  )
}
