import { useEffect, useState } from 'react'
import type { FlashSaleSection as FlashSaleSectionType } from '@/types'
import { useActiveFlashSales } from '@/features/promotions/hooks'
import { useProductsByIds } from '@/features/products/hooks'
import { ProductGrid } from '@/components/ecommerce/ProductGrid'
import { Badge } from '@/components/ui/Badge'

function useCountdown(endsAt: string): string {
  const [label, setLabel] = useState('')
  useEffect(() => {
    function tick() {
      const remaining = new Date(endsAt).getTime() - Date.now()
      if (remaining <= 0) { setLabel('Ended'); return }
      const hours = Math.floor(remaining / 3_600_000)
      const minutes = Math.floor((remaining % 3_600_000) / 60_000)
      const seconds = Math.floor((remaining % 60_000) / 1000)
      setLabel(`${hours}h ${minutes}m ${seconds}s`)
    }
    tick()
    const interval = setInterval(tick, 1000)
    return () => clearInterval(interval)
  }, [endsAt])
  return label
}

export function FlashSaleSection({ section }: { section: FlashSaleSectionType }) {
  const { data: flashSales = [] } = useActiveFlashSales()
  const sale = flashSales.find(f => f.id === section.flashSaleId)
  const { data: products = [] } = useProductsByIds(sale?.productIds ?? [])
  const countdown = useCountdown(sale?.endsAt ?? new Date().toISOString())

  if (!sale) return null

  return (
    <section>
      <div className="mb-6 flex items-center gap-3">
        <h2 className="text-xl font-semibold tracking-tight sm:text-2xl">{sale.title}</h2>
        <Badge variant="sale">-{sale.discountPercent}%</Badge>
        <span className="text-sm font-medium tabular-nums text-muted-foreground" aria-live="polite">Ends in {countdown}</span>
      </div>
      <ProductGrid products={products} />
    </section>
  )
}
