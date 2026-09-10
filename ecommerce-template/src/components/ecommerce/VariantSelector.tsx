import { cn } from '@/lib/utils'
import type { Product, ProductVariant } from '@/types'

interface VariantSelectorProps {
  product: Product
  selected: Record<string, string>
  onChange: (optionName: string, value: string) => void
}

export function VariantSelector({ product, selected, onChange }: VariantSelectorProps) {
  if (product.variantOptionNames.length === 0) return null

  return (
    <div className="space-y-4">
      {product.variantOptionNames.map(optionName => {
        const values = Array.from(new Set(product.variants.flatMap(v => v.options.filter(o => o.name === optionName).map(o => o.value))))
        return (
          <div key={optionName}>
            <p className="mb-2 text-sm font-medium">{optionName}: <span className="font-normal text-muted-foreground">{selected[optionName] ?? 'Select'}</span></p>
            <div className="flex flex-wrap gap-2" role="radiogroup" aria-label={optionName}>
              {values.map(value => {
                const isSelected = selected[optionName] === value
                const available = product.variants.some(
                  v => v.options.some(o => o.name === optionName && o.value === value) && v.stock > 0
                    && Object.entries(selected).every(([k, val]) => k === optionName || v.options.some(o => o.name === k && o.value === val)),
                )
                return (
                  <button
                    key={value}
                    type="button"
                    role="radio"
                    aria-checked={isSelected}
                    disabled={!available}
                    onClick={() => onChange(optionName, value)}
                    className={cn(
                      'rounded-[var(--radius)] border px-3 py-1.5 text-sm transition-colors',
                      isSelected ? 'border-primary bg-primary text-primary-foreground' : 'border-input hover:bg-secondary',
                      !available && 'cursor-not-allowed opacity-40 line-through',
                    )}
                  >
                    {value}
                  </button>
                )
              })}
            </div>
          </div>
        )
      })}
    </div>
  )
}

export function findMatchingVariant(product: Product, selected: Record<string, string>): ProductVariant | undefined {
  if (product.variantOptionNames.some(name => !selected[name])) return undefined
  return product.variants.find(v => v.options.every(o => selected[o.name] === o.value))
}
