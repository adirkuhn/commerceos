import { useState } from 'react'
import { Truck, Store } from 'lucide-react'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'

const PICKUP_LOCATIONS = ['Boulder, CO', 'Denver, CO', 'Fort Collins, CO']

export function ShippingEstimator() {
  const [postalCode, setPostalCode] = useState('')
  const [estimate, setEstimate] = useState<string | null>(null)

  return (
    <div className="space-y-4 rounded-[var(--radius)] border border-border p-4">
      <div className="flex items-center gap-2 text-sm font-medium"><Truck className="h-4 w-4" /> Delivery estimate</div>
      <form
        className="flex gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          if (!postalCode.trim()) return
          const days = 3 + (postalCode.trim().length % 4)
          setEstimate(`Arrives in ${days}-${days + 2} business days to ${postalCode}`)
        }}
      >
        <Input placeholder="Enter postal code" value={postalCode} onChange={(e) => setPostalCode(e.target.value)} aria-label="Postal code for delivery estimate" />
        <Button type="submit" variant="outline">Estimate</Button>
      </form>
      {estimate && <p className="text-sm text-muted-foreground">{estimate}</p>}

      <div className="flex items-center gap-2 text-sm font-medium"><Store className="h-4 w-4" /> Store pickup available</div>
      <ul className="space-y-1 text-sm text-muted-foreground">
        {PICKUP_LOCATIONS.map(loc => <li key={loc}>{loc} -- ready in 2 hours</li>)}
      </ul>
    </div>
  )
}
