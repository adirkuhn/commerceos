import { useTranslation } from 'react-i18next'

// The ?discount= percentage, shared by both tabs.
export function DiscountField({ value, onChange }: { value: string; onChange: (value: string | null) => void }) {
  const { t } = useTranslation('priceList')
  return (
    <label className="flex items-center gap-2 text-xs text-ink-3" title={t('discount.hint')}>
      {t('discount.label')}
      <span className="relative">
        <input
          type="number" min={0} max={100} step="0.5" inputMode="decimal"
          value={value}
          onChange={e => onChange(e.target.value)}
          aria-label={t('discount.label')}
          className="w-20 pl-2 pr-6 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus tabular-nums"
        />
        <span className="absolute right-2 top-1/2 -translate-y-1/2 text-ink-4">%</span>
      </span>
    </label>
  )
}
