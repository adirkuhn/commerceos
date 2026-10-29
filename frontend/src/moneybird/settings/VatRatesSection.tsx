import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { inputClass, labelClass, SettingsSection } from '../../shipping/postnl/settingsSectionUi'
import type { MoneybirdSetup, TaxRateMapping } from '../types'
import { SectionFooter } from './SectionFooter'
import { useSectionSave, type SaveSettings } from './useSectionSave'

const percent = (basisPoints: number) => (basisPoints / 100).toString()

interface Props {
  setup: MoneybirdSetup
  // The shop's own VAT rates (Settings -> Tax), offered for mapping.
  shopRates: number[]
  onSave: SaveSettings
}

// One row per VAT rate -- the shop's own plus any already mapped -- each
// with the Moneybird tax rate it is booked with.
export function VatRatesSection({ setup, shopRates, onSave }: Props) {
  const { t } = useTranslation('integrations')
  const [mapped, setMapped] = useState<Record<number, string>>(() =>
    Object.fromEntries(setup.settings.tax_rates.map(m => [m.rate_basis_points, m.moneybird_tax_rate_id])))
  const [extra, setExtra] = useState<number[]>([])
  const [newRate, setNewRate] = useState('')
  const save = useSectionSave(onSave)

  const rows = [...new Set([...shopRates, ...setup.settings.tax_rates.map(m => m.rate_basis_points), ...extra])].sort((a, b) => b - a)

  const addRate = () => {
    const basisPoints = Math.round(Number(newRate.replace(',', '.')) * 100)
    if (!Number.isFinite(basisPoints) || basisPoints < 0 || basisPoints > 10000 || newRate.trim() === '') return
    setExtra(prev => [...prev, basisPoints])
    setNewRate('')
  }

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    const taxRates: TaxRateMapping[] = rows
      .filter(rate => mapped[rate])
      .map(rate => ({ rate_basis_points: rate, moneybird_tax_rate_id: mapped[rate] }))
    void save.run({ tax_rates: taxRates })
  }

  if (!setup.settings.administration_id) {
    return (
      <SettingsSection id="mb-vat" title={t('moneybird.vat.title')} body={t('moneybird.vat.body')}>
        <p className="text-2xs text-ink-4">{t('moneybird.vat.needsAdministration')}</p>
      </SettingsSection>
    )
  }

  return (
    <SettingsSection id="mb-vat" title={t('moneybird.vat.title')} body={t('moneybird.vat.body')}>
      <form onSubmit={submit} className="space-y-3">
        {rows.map(rate => (
          <div key={rate} className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <label htmlFor={`mb-vat-${rate}`} className="w-32 text-xs text-ink-2">{t('moneybird.vat.rate', { rate: percent(rate) })}</label>
            <select id={`mb-vat-${rate}`} value={mapped[rate] ?? ''} className={`${inputClass} flex-1 min-w-[10rem] w-auto`}
              onChange={e => setMapped(prev => ({ ...prev, [rate]: e.target.value }))}>
              <option value="">{t('moneybird.vat.notMapped')}</option>
              {setup.tax_rates.map(r => <option key={r.id} value={r.id}>{r.name} ({r.percentage}%)</option>)}
            </select>
          </div>
        ))}
        <div className="flex items-end gap-2">
          <div>
            <label htmlFor="mb-vat-new" className={labelClass}>{t('moneybird.vat.addRate')}</label>
            <input id="mb-vat-new" inputMode="decimal" value={newRate} onChange={e => setNewRate(e.target.value)} className={`${inputClass} w-24`} />
          </div>
          <button type="button" onClick={addRate} className="px-3 py-1.5 rounded-lg border border-line text-xs text-ink-2 hover:bg-hover">
            {t('moneybird.vat.add')}
          </button>
        </div>
        <SectionFooter {...save} />
      </form>
    </SettingsSection>
  )
}
