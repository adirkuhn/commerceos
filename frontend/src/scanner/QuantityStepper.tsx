import { Minus, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'

// min/max are only passed for a quantity that draws down a specific row's
// available stock (Move, Damaged/Lost, Discard) -- both the stepper buttons
// and free typing get clamped into range there, plus a Max button to fill
// it in one tap. Left undefined (Adjust's quantity-change field) the
// stepper is unbounded either direction, since a recount correction can go
// either way.
export function QuantityStepper({ id, label, value, onChange, min, max }: { id: string; label: string; value: string; onChange: (v: string) => void; min?: number; max?: number }) {
  const { t } = useTranslation('scanner')
  const n = Number(value) || 0
  function clamp(v: number): number {
    let out = v
    if (min !== undefined) out = Math.max(min, out)
    if (max !== undefined) out = Math.min(max, out)
    return out
  }
  return (
    <div>
      <div className="flex items-center justify-between mb-1.5">
        <label htmlFor={id} className="block text-xs text-gray-400">{label}</label>
        {max !== undefined && (
          <button type="button" onClick={() => onChange(String(clamp(max)))} className="text-xs font-medium text-accent">
            {t('stepper.max')}
          </button>
        )}
      </div>
      <div className="flex items-center gap-2">
        <button type="button" onClick={() => onChange(String(clamp(n - 1)))} className="w-11 h-11 shrink-0 rounded-xl bg-night-800 flex items-center justify-center" aria-label={t('stepper.decrease', { label })}>
          <Minus size={14} />
        </button>
        <input
          id={id}
          inputMode="numeric"
          value={value}
          onChange={e => {
            const digits = e.target.value.replace(/(?!^-)[^\d]/g, '')
            onChange(digits === '' || digits === '-' ? digits : String(clamp(Number(digits))))
          }}
          className="flex-1 min-w-0 px-2 py-2 text-center text-base rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
        />
        <button type="button" onClick={() => onChange(String(clamp(n + 1)))} className="w-11 h-11 shrink-0 rounded-xl bg-night-800 flex items-center justify-center" aria-label={t('stepper.increase', { label })}>
          <Plus size={14} />
        </button>
      </div>
    </div>
  )
}
