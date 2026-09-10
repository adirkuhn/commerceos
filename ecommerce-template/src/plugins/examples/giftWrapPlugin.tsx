import { useState } from 'react'
import type { EcommercePlugin } from '@/types'
import { Checkbox } from '@/components/ui/Checkbox'

// Proves CheckoutExtension end-to-end: a real, interactive checkout-step
// addition a plugin author could ship without touching CheckoutPage.tsx
// itself -- see features/checkout's ShippingStep, which renders
// pluginRegistry.checkoutExtensions('shipping').
function GiftWrapOption() {
  const [checked, setChecked] = useState(false)
  return (
    <div className="flex items-start gap-3 rounded-lg border border-border bg-card p-4">
      <Checkbox checked={checked} onCheckedChange={(v) => setChecked(v === true)} id="gift-wrap" />
      <label htmlFor="gift-wrap" className="text-sm">
        <span className="font-medium">Add gift wrap (+$4.00)</span>
        <p className="text-muted-foreground">Wrapped in kraft paper with a card -- no price shown inside.</p>
      </label>
    </div>
  )
}

export const giftWrapPlugin: EcommercePlugin = {
  name: 'gift-wrap',
  checkoutExtensions: [{ id: 'gift-wrap', step: 'shipping', render: GiftWrapOption }],
}
