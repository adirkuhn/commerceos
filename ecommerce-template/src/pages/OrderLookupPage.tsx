import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Container } from '@/components/layout/Container'
import { Seo } from '@/components/seo/Seo'
import { Input } from '@/components/ui/Input'
import { Label } from '@/components/ui/Label'
import { Button } from '@/components/ui/Button'
import { findOrderByNumberAndEmail } from '@/api/orders'
import type { Order } from '@/types'

export default function OrderLookupPage() {
  const [orderNumber, setOrderNumber] = useState('')
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<Order | null | undefined>(undefined)

  return (
    <>
      <Seo title="Track your order" />
      <Container className="max-w-md py-16">
        <h1 className="mb-1 text-2xl font-semibold tracking-tight">Track your order</h1>
        <p className="mb-6 text-sm text-muted-foreground">Try order <strong>AU-10231</strong> with <strong>morgan@example.com</strong>.</p>

        <form
          className="space-y-4"
          onSubmit={async (e) => {
            e.preventDefault()
            setLoading(true)
            const order = await findOrderByNumberAndEmail(orderNumber, email)
            setResult(order)
            setLoading(false)
          }}
        >
          <div>
            <Label htmlFor="lookup-order-number">Order number</Label>
            <Input id="lookup-order-number" required value={orderNumber} onChange={(e) => setOrderNumber(e.target.value)} className="mt-1.5" placeholder="AU-10231" />
          </div>
          <div>
            <Label htmlFor="lookup-email">Email</Label>
            <Input id="lookup-email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} className="mt-1.5" />
          </div>
          <Button type="submit" className="w-full" disabled={loading}>{loading ? 'Looking up…' : 'Track order'}</Button>
        </form>

        {result === null && <p role="alert" className="mt-4 text-sm text-destructive">No order found matching that order number and email.</p>}
        {result && (
          <p className="mt-4 rounded-[var(--radius)] bg-accent p-4 text-sm text-accent-foreground">
            Found it! <Link to={`/order/${result.id}`} className="font-medium underline">View order {result.orderNumber}</Link>
          </p>
        )}
      </Container>
    </>
  )
}
