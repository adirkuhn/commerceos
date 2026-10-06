import { RefreshCw } from 'lucide-react'
import clsx from 'clsx'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { Supplier } from '../suppliers/types'
import { dayOptions, type DayRange } from './planningWindow'

export const SELECT = 'px-2 py-1.5 rounded-lg border border-line bg-raised text-xs text-ink focus:outline-none focus:ring-1 focus:ring-accent'

export function Field({ id, label, children }: { id: string; label: string; children: ReactNode }) {
  return (
    <label className="flex flex-col gap-1 text-2xs text-ink-4" htmlFor={id}>
      {label}
      {children}
    </label>
  )
}

interface DaysPickerProps {
  id: string
  label: string
  range: DayRange
  value: number
  onChange: (days: number) => void
}

export function DaysPicker({ id, label, range, value, onChange }: DaysPickerProps) {
  const { t } = useTranslation('purchasePlanning')
  return (
    <Field id={id} label={label}>
      <select id={id} className={SELECT} value={value} onChange={e => onChange(Number(e.target.value))}>
        {dayOptions(range, value).map(d => <option key={d} value={d}>{t('window.days', { count: d })}</option>)}
      </select>
    </Field>
  )
}

interface SupplierFilterProps {
  id: string
  suppliers: Supplier[]
  value: string | null
  onChange: (supplierId: string | null) => void
}

export function SupplierFilter({ id, suppliers, value, onChange }: SupplierFilterProps) {
  const { t } = useTranslation('purchasePlanning')
  return (
    <Field id={id} label={t('window.supplier')}>
      <select id={id} className={SELECT} value={value ?? ''} onChange={e => onChange(e.target.value || null)}>
        <option value="">{t('window.allSuppliers')}</option>
        {suppliers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
      </select>
    </Field>
  )
}

export function RefreshButton({ loading, onClick }: { loading: boolean; onClick: () => void }) {
  const { t } = useTranslation('purchasePlanning')
  return (
    <button
      onClick={onClick}
      disabled={loading}
      className="ml-auto flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-raised text-xs text-ink-2 hover:bg-hover transition-colors font-medium disabled:opacity-50"
    >
      <RefreshCw size={13} className={clsx(loading && 'animate-spin')} /> {t('refresh')}
    </button>
  )
}

interface PlanResultProps {
  loading: boolean
  error: string | null
  empty: boolean
  emptyText: string
  children: ReactNode
}

// Loading / error / empty states around a plan's supplier cards.
export function PlanResult({ loading, error, empty, emptyText, children }: PlanResultProps) {
  const { t } = useTranslation('purchasePlanning')
  if (error) return <p role="alert" className="text-xs text-danger">{error}</p>
  if (empty) return loading ? <p className="text-xs text-ink-4">{t('loading')}</p> : <p className="py-10 text-center text-xs text-ink-4">{emptyText}</p>
  return <>{children}</>
}
