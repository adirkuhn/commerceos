import { X } from 'lucide-react'
import type { SearchFilters } from '@/types'
import { useBrands } from '@/features/categories/hooks'
import { Checkbox } from '@/components/ui/Checkbox'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { Button } from '@/components/ui/Button'
import { Badge } from '@/components/ui/Badge'

export interface FilterState {
  brandId?: string
  minPrice?: number
  maxPrice?: number
  minRating?: number
  inStockOnly?: boolean
  onSaleOnly?: boolean
}

const SORT_OPTIONS: { value: NonNullable<SearchFilters['sort']>; label: string }[] = [
  { value: 'relevance', label: 'Relevance' },
  { value: 'newest', label: 'Newest' },
  { value: 'price_asc', label: 'Price: low to high' },
  { value: 'price_desc', label: 'Price: high to low' },
  { value: 'rating', label: 'Highest rated' },
  { value: 'best_selling', label: 'Best selling' },
]

export function SortSelect({ value, onChange }: { value: SearchFilters['sort']; onChange: (sort: SearchFilters['sort']) => void }) {
  return (
    <Select aria-label="Sort by" value={value ?? 'relevance'} onChange={(e) => onChange(e.target.value as SearchFilters['sort'])} className="w-44">
      {SORT_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
    </Select>
  )
}

export function ProductFilterSidebar({ filters, onChange }: { filters: FilterState; onChange: (next: FilterState) => void }) {
  const { data: brands = [] } = useBrands()

  return (
    <div className="space-y-6">
      <div>
        <h3 className="mb-3 text-sm font-semibold">Price</h3>
        <div className="flex items-center gap-2">
          <Input
            type="number" min={0} placeholder="Min" aria-label="Minimum price"
            value={filters.minPrice !== undefined ? filters.minPrice / 100 : ''}
            onChange={(e) => onChange({ ...filters, minPrice: e.target.value ? Number(e.target.value) * 100 : undefined })}
          />
          <span className="text-muted-foreground">–</span>
          <Input
            type="number" min={0} placeholder="Max" aria-label="Maximum price"
            value={filters.maxPrice !== undefined ? filters.maxPrice / 100 : ''}
            onChange={(e) => onChange({ ...filters, maxPrice: e.target.value ? Number(e.target.value) * 100 : undefined })}
          />
        </div>
      </div>

      <div>
        <h3 className="mb-3 text-sm font-semibold">Brand</h3>
        <div className="space-y-2">
          {brands.map(b => (
            <label key={b.id} className="flex items-center gap-2 text-sm">
              <Checkbox
                checked={filters.brandId === b.id}
                onCheckedChange={(checked) => onChange({ ...filters, brandId: checked ? b.id : undefined })}
              />
              {b.name}
            </label>
          ))}
        </div>
      </div>

      <div>
        <h3 className="mb-3 text-sm font-semibold">Rating</h3>
        <div className="space-y-2">
          {[4, 3].map(rating => (
            <label key={rating} className="flex items-center gap-2 text-sm">
              <Checkbox
                checked={filters.minRating === rating}
                onCheckedChange={(checked) => onChange({ ...filters, minRating: checked ? rating : undefined })}
              />
              {rating}+ stars
            </label>
          ))}
        </div>
      </div>

      <div>
        <h3 className="mb-3 text-sm font-semibold">Availability</h3>
        <label className="flex items-center gap-2 text-sm">
          <Checkbox checked={!!filters.inStockOnly} onCheckedChange={(checked) => onChange({ ...filters, inStockOnly: checked })} />
          In stock only
        </label>
        <label className="mt-2 flex items-center gap-2 text-sm">
          <Checkbox checked={!!filters.onSaleOnly} onCheckedChange={(checked) => onChange({ ...filters, onSaleOnly: checked })} />
          On sale
        </label>
      </div>

      <Button variant="ghost" size="sm" onClick={() => onChange({})}>Clear all filters</Button>
    </div>
  )
}

export function ActiveFilterChips({ filters, onChange, brandLabel }: { filters: FilterState; onChange: (next: FilterState) => void; brandLabel?: string }) {
  const chips: { key: keyof FilterState; label: string }[] = []
  if (filters.brandId) chips.push({ key: 'brandId', label: brandLabel ?? 'Brand' })
  if (filters.minPrice !== undefined) chips.push({ key: 'minPrice', label: `Min $${filters.minPrice / 100}` })
  if (filters.maxPrice !== undefined) chips.push({ key: 'maxPrice', label: `Max $${filters.maxPrice / 100}` })
  if (filters.minRating !== undefined) chips.push({ key: 'minRating', label: `${filters.minRating}+ stars` })
  if (filters.inStockOnly) chips.push({ key: 'inStockOnly', label: 'In stock' })
  if (filters.onSaleOnly) chips.push({ key: 'onSaleOnly', label: 'On sale' })

  if (chips.length === 0) return null

  return (
    <div className="flex flex-wrap gap-2">
      {chips.map(chip => (
        <Badge key={chip.key} variant="secondary" className="gap-1">
          {chip.label}
          <button aria-label={`Remove filter ${chip.label}`} onClick={() => onChange({ ...filters, [chip.key]: undefined })}>
            <X className="h-3 w-3" />
          </button>
        </Badge>
      ))}
    </div>
  )
}
