import { useState } from 'react'
import { Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { inputClass, labelClass, SettingsSection } from '../../shipping/postnl/settingsSectionUi'
import type { MoneybirdPaymentMethod, MoneybirdSetup, PaymentMethodMapping } from '../types'
import { SectionFooter } from './SectionFooter'
import { useSectionSave, type SaveSettings } from './useSectionSave'

// Order payment methods CommerceOS sees most (Woo gateways, POS), offered
// as suggestions -- any other can be typed.
const KNOWN_METHODS = ['bacs', 'cod', 'cheque', 'stripe', 'paypal', 'mollie_wc_gateway_ideal', 'cash', 'pin']

// Which Moneybird payment method each order payment method is registered as.
export function PaymentMethodsSection({ setup, onSave }: { setup: MoneybirdSetup; onSave: SaveSettings }) {
  const { t } = useTranslation('integrations')
  const [rows, setRows] = useState<PaymentMethodMapping[]>(setup.settings.payment_methods)
  const save = useSectionSave(onSave)
  const firstMethod = setup.payment_methods[0] ?? 'bank_transfer'

  const update = (i: number, patch: Partial<PaymentMethodMapping>) =>
    setRows(prev => prev.map((r, j) => (j === i ? { ...r, ...patch } : r)))

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    void save.run({ payment_methods: rows.filter(r => r.method.trim() !== '').map(r => ({ ...r, method: r.method.trim() })) })
  }

  return (
    <SettingsSection id="mb-payments" title={t('moneybird.payments.title')} body={t('moneybird.payments.body')}>
      <form onSubmit={submit} className="space-y-3">
        <datalist id="mb-known-methods">{KNOWN_METHODS.map(m => <option key={m} value={m} />)}</datalist>
        {rows.map((row, i) => (
          <div key={i} className="flex flex-wrap items-end gap-2">
            <div className="flex-1 min-w-[10rem]">
              <label htmlFor={`mb-pm-method-${i}`} className={labelClass}>{t('moneybird.payments.method')}</label>
              <input id={`mb-pm-method-${i}`} list="mb-known-methods" value={row.method} required
                onChange={e => update(i, { method: e.target.value })} className={inputClass} />
            </div>
            <div className="flex-1 min-w-[10rem]">
              <label htmlFor={`mb-pm-moneybird-${i}`} className={labelClass}>{t('moneybird.payments.moneybirdMethod')}</label>
              <select id={`mb-pm-moneybird-${i}`} value={row.moneybird_payment_method} className={inputClass}
                onChange={e => update(i, { moneybird_payment_method: e.target.value as MoneybirdPaymentMethod })}>
                {setup.payment_methods.map(m => <option key={m} value={m}>{t(`moneybird.payments.methods.${m}`)}</option>)}
              </select>
            </div>
            <button type="button" aria-label={t('moneybird.payments.remove')} title={t('moneybird.payments.remove')}
              onClick={() => setRows(prev => prev.filter((_, j) => j !== i))}
              className="p-2 rounded-lg text-ink-4 hover:bg-hover hover:text-danger">
              <Trash2 size={14} aria-hidden />
            </button>
          </div>
        ))}
        <button type="button" onClick={() => setRows(prev => [...prev, { method: '', moneybird_payment_method: firstMethod }])}
          className="px-3 py-1.5 rounded-lg border border-line text-xs text-ink-2 hover:bg-hover">
          {t('moneybird.payments.add')}
        </button>
        <SectionFooter {...save} />
      </form>
    </SettingsSection>
  )
}
