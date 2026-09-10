import { Minus, Plus } from 'lucide-react'
import { Button } from '@/components/ui/Button'

export function QuantitySelector({
  value, onChange, min = 1, max = 99, disabled,
}: { value: number; onChange: (value: number) => void; min?: number; max?: number; disabled?: boolean }) {
  return (
    <div className="inline-flex items-center rounded-[var(--radius)] border border-input">
      <Button
        type="button" variant="ghost" size="icon" className="h-9 w-9"
        disabled={disabled || value <= min}
        onClick={() => onChange(Math.max(min, value - 1))}
        aria-label="Decrease quantity"
      >
        <Minus className="h-3.5 w-3.5" />
      </Button>
      <span className="w-8 text-center text-sm tabular-nums" aria-live="polite">{value}</span>
      <Button
        type="button" variant="ghost" size="icon" className="h-9 w-9"
        disabled={disabled || value >= max}
        onClick={() => onChange(Math.min(max, value + 1))}
        aria-label="Increase quantity"
      >
        <Plus className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}
