import { Scale } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { useCompareStore, MAX_COMPARE_ITEMS } from '@/stores/compareStore'
import { toast } from '@/components/ui/toast'
import { cn } from '@/lib/utils'

export function CompareButton({ productId, className }: { productId: string; className?: string }) {
  const comparing = useCompareStore(s => s.isComparing(productId))
  const count = useCompareStore(s => s.productIds.length)
  const toggle = useCompareStore(s => s.toggle)

  return (
    <Button
      type="button" variant="outline" size="icon"
      aria-pressed={comparing}
      aria-label={comparing ? 'Remove from comparison' : 'Add to comparison'}
      onClick={() => {
        if (!comparing && count >= MAX_COMPARE_ITEMS) {
          toast({ title: `You can compare up to ${MAX_COMPARE_ITEMS} products`, variant: 'destructive' })
          return
        }
        toggle(productId)
      }}
      className={className}
    >
      <Scale className={cn('h-4 w-4', comparing && 'text-primary')} />
    </Button>
  )
}
