import type { ShippingMethod, ShippingMethodId } from '@/types'
import { RadioGroup, RadioGroupItem } from '@/components/ui/RadioGroup'
import { formatMoney } from '@/lib/utils'

export const SHIPPING_METHODS: ShippingMethod[] = [
  { id: 'standard', label: 'Standard shipping', description: 'Delivered by carrier', price: 900, etaLabel: '5-7 business days' },
  { id: 'express', label: 'Express shipping', description: 'Delivered by carrier, faster', price: 2400, etaLabel: '2 business days' },
  { id: 'pickup', label: 'Store pickup', description: 'Pick up at our Boulder, CO location', price: 0, etaLabel: 'Ready in 2 hours' },
]

export function ShippingMethodSelector({
  value, onChange, currency,
}: { value: ShippingMethodId; onChange: (id: ShippingMethodId) => void; currency: string }) {
  return (
    <RadioGroup value={value} onValueChange={(v) => onChange(v as ShippingMethodId)} name="shipping-method">
      {SHIPPING_METHODS.map(method => (
        <label
          key={method.id}
          htmlFor={`shipping-${method.id}`}
          className="flex cursor-pointer items-center justify-between gap-3 rounded-[var(--radius)] border border-border p-4 has-[:checked]:border-primary has-[:checked]:bg-accent"
        >
          <div className="flex items-center gap-3">
            <RadioGroupItem id={`shipping-${method.id}`} value={method.id} />
            <div>
              <p className="text-sm font-medium">{method.label}</p>
              <p className="text-xs text-muted-foreground">{method.description} · {method.etaLabel}</p>
            </div>
          </div>
          <p className="text-sm font-medium">{method.price === 0 ? 'Free' : formatMoney(method.price, currency)}</p>
        </label>
      ))}
    </RadioGroup>
  )
}
