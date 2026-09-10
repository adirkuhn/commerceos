import { useState, useEffect } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Container } from '@/components/layout/Container'
import { Seo } from '@/components/seo/Seo'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Label } from '@/components/ui/Label'
import { Checkbox } from '@/components/ui/Checkbox'
import { Separator } from '@/components/ui/Separator'
import { EmptyState, LoadingState } from '@/components/ui/States'
import { AddressForm } from '@/components/ecommerce/AddressForm'
import { ShippingMethodSelector } from '@/components/ecommerce/ShippingMethodSelector'
import { PaymentMethodSelector } from '@/components/ecommerce/PaymentMethodSelector'
import { CheckoutSummary } from '@/components/ecommerce/CheckoutSummary'
import { checkoutCustomerSchema, type CheckoutCustomerFormValues } from '@/lib/validation'
import { useCart } from '@/features/cart/hooks'
import { usePlaceOrder } from '@/features/checkout/hooks'
import { useCheckoutStore, CHECKOUT_STEPS, type CheckoutStep } from '@/stores/checkoutStore'
import { useAuthStore } from '@/stores/authStore'
import { pluginRegistry } from '@/plugins'
import { analytics } from '@/services/analytics'
import { cn } from '@/lib/utils'
import { ShoppingBag } from 'lucide-react'

const STEP_LABELS: Record<CheckoutStep, string> = { customer: 'Customer', shipping: 'Shipping', payment: 'Payment', review: 'Review' }

export default function CheckoutPage() {
  const navigate = useNavigate()
  const { data: cart, isLoading } = useCart()
  const session = useAuthStore(s => s.session)
  const checkout = useCheckoutStore()
  const placeOrder = usePlaceOrder()
  const [orderError, setOrderError] = useState<string | null>(null)

  useEffect(() => { analytics.track('checkout_started') }, [])

  if (isLoading) return <LoadingState />
  if (!cart || cart.items.length === 0) {
    return (
      <Container className="py-16">
        <EmptyState
          icon={<ShoppingBag className="h-8 w-8" />}
          title="Your cart is empty"
          description="Add something to your cart before checking out."
          action={<Button asChild className="mt-2"><Link to="/">Continue shopping</Link></Button>}
        />
      </Container>
    )
  }

  const stepIndex = CHECKOUT_STEPS.indexOf(checkout.step)

  return (
    <>
      <Seo title="Checkout" />
      <Container className="grid gap-10 py-8 lg:grid-cols-[1fr_20rem]">
        <div>
          <ol className="mb-8 flex flex-wrap gap-4">
            {CHECKOUT_STEPS.map((step, i) => (
              <li key={step} className={cn('flex items-center gap-2 text-sm', i <= stepIndex ? 'font-medium text-foreground' : 'text-muted-foreground')}>
                <span className={cn('flex h-6 w-6 items-center justify-center rounded-full text-xs', i <= stepIndex ? 'bg-primary text-primary-foreground' : 'bg-secondary')}>{i + 1}</span>
                {STEP_LABELS[step]}
              </li>
            ))}
          </ol>

          {checkout.step === 'customer' && <CustomerStep isAuthenticated={!!session} />}
          {checkout.step === 'shipping' && <ShippingStep currency={cart.currency} />}
          {checkout.step === 'payment' && <PaymentStep />}
          {checkout.step === 'review' && (
            <ReviewStep
              onPlaceOrder={async () => {
                if (!checkout.billingAddress || !checkout.shippingAddress) return
                setOrderError(null)
                try {
                  const order = await placeOrder.mutateAsync({
                    cart,
                    input: {
                      email: checkout.email, billingAddress: checkout.billingAddress, shippingAddress: checkout.shippingAddress,
                      shippingMethodId: checkout.shippingMethodId, paymentMethodId: checkout.paymentMethodId, customerId: session?.customer.id,
                    },
                  })
                  checkout.reset()
                  navigate(`/order-confirmation/${order.id}`)
                } catch (err) {
                  setOrderError(err instanceof Error ? err.message : 'Could not place your order.')
                }
              }}
              placing={placeOrder.isPending}
              error={orderError}
            />
          )}
        </div>

        <aside className="h-fit space-y-4 rounded-[var(--radius)] border border-border p-5">
          <h2 className="text-sm font-semibold">Order summary</h2>
          <Separator />
          <ul className="max-h-64 space-y-3 overflow-y-auto">
            {cart.items.map(item => (
              <li key={item.id} className="flex gap-3">
                {item.imageUrl && <img src={item.imageUrl} alt="" className="h-12 w-12 rounded-[var(--radius)] object-cover" />}
                <div className="flex-1 text-sm">
                  <p className="font-medium">{item.name} <span className="text-muted-foreground">× {item.quantity}</span></p>
                  {item.variantLabel && <p className="text-xs text-muted-foreground">{item.variantLabel}</p>}
                </div>
              </li>
            ))}
          </ul>
          <Separator />
          <CheckoutSummary totals={cart.totals} currency={cart.currency} />
        </aside>
      </Container>
    </>
  )
}

function CustomerStep({ isAuthenticated }: { isAuthenticated: boolean }) {
  const checkout = useCheckoutStore()
  const { register, handleSubmit, formState: { errors } } = useForm<CheckoutCustomerFormValues>({
    resolver: zodResolver(checkoutCustomerSchema), defaultValues: { email: checkout.email },
  })

  return (
    <div className="max-w-md space-y-6">
      {!isAuthenticated && (
        <p className="text-sm text-muted-foreground">
          <Link to="/login?next=/checkout" className="text-primary underline">Log in</Link> for a faster checkout, or continue as a guest below.
        </p>
      )}
      <form noValidate onSubmit={handleSubmit((values) => { checkout.setEmail(values.email); checkout.nextStep() })} className="space-y-4">
        <div>
          <Label htmlFor="checkout-email">Email</Label>
          <Input id="checkout-email" type="email" className="mt-1.5" {...register('email')} />
          {errors.email && <p className="mt-1 text-xs text-destructive">{errors.email.message}</p>}
        </div>
        <div>
          <Label htmlFor="checkout-phone">Phone (optional)</Label>
          <Input id="checkout-phone" type="tel" className="mt-1.5" {...register('phone')} />
        </div>
        <Button type="submit">Continue to shipping</Button>
      </form>
    </div>
  )
}

function ShippingStep({ currency }: { currency: string }) {
  const checkout = useCheckoutStore()
  const shippingExtensions = pluginRegistry.checkoutExtensions('shipping')

  return (
    <div className="max-w-md space-y-8">
      <div>
        <h2 className="mb-3 text-sm font-semibold">Shipping address</h2>
        <AddressForm idPrefix="shipping" defaultValues={checkout.shippingAddress ?? undefined} submitLabel="Continue" onSubmit={(values) => { checkout.setShippingAddress({ id: 'shipping', ...values }); if (checkout.shippingAddressSameAsBilling) checkout.setBillingAddress({ id: 'billing', ...values }) }} />
      </div>
      <label className="flex items-center gap-2 text-sm">
        <Checkbox checked={checkout.shippingAddressSameAsBilling} onCheckedChange={checkout.setShippingAddressSameAsBilling} />
        Billing address same as shipping
      </label>
      {!checkout.shippingAddressSameAsBilling && (
        <div>
          <h2 className="mb-3 text-sm font-semibold">Billing address</h2>
          <AddressForm idPrefix="billing" showVatField defaultValues={checkout.billingAddress ?? undefined} submitLabel="Save billing address" onSubmit={(values) => checkout.setBillingAddress({ id: 'billing', ...values })} />
        </div>
      )}
      <div>
        <h2 className="mb-3 text-sm font-semibold">Shipping method</h2>
        <ShippingMethodSelector value={checkout.shippingMethodId} onChange={checkout.setShippingMethodId} currency={currency} />
      </div>
      {shippingExtensions.map(ext => <ext.render key={ext.id} />)}
      <Button disabled={!checkout.shippingAddress} onClick={() => checkout.nextStep()}>Continue to payment</Button>
    </div>
  )
}

function PaymentStep() {
  const checkout = useCheckoutStore()
  return (
    <div className="max-w-md space-y-6">
      <h2 className="text-sm font-semibold">Payment method</h2>
      <PaymentMethodSelector value={checkout.paymentMethodId} onChange={checkout.setPaymentMethodId} />
      <p className="text-xs text-muted-foreground">This is a demo storefront -- no real payment is processed regardless of the method chosen.</p>
      <Button onClick={() => checkout.nextStep()}>Review order</Button>
    </div>
  )
}

function ReviewStep({ onPlaceOrder, placing, error }: { onPlaceOrder: () => void; placing: boolean; error: string | null }) {
  const checkout = useCheckoutStore()
  const reviewExtensions = pluginRegistry.checkoutExtensions('review')

  return (
    <div className="max-w-md space-y-6">
      <div>
        <h2 className="mb-1 text-sm font-semibold">Contact</h2>
        <p className="text-sm text-muted-foreground">{checkout.email}</p>
      </div>
      {checkout.shippingAddress && (
        <div>
          <h2 className="mb-1 text-sm font-semibold">Shipping to</h2>
          <p className="text-sm text-muted-foreground">
            {checkout.shippingAddress.firstName} {checkout.shippingAddress.lastName}, {checkout.shippingAddress.line1}, {checkout.shippingAddress.city} {checkout.shippingAddress.postalCode}, {checkout.shippingAddress.country}
          </p>
        </div>
      )}
      <div>
        <h2 className="mb-1 text-sm font-semibold">Shipping method</h2>
        <p className="text-sm capitalize text-muted-foreground">{checkout.shippingMethodId}</p>
      </div>
      <div>
        <h2 className="mb-1 text-sm font-semibold">Payment method</h2>
        <p className="text-sm capitalize text-muted-foreground">{checkout.paymentMethodId.replace('_', ' ')}</p>
      </div>
      {reviewExtensions.map(ext => <ext.render key={ext.id} />)}
      <label className="flex items-start gap-2 text-sm">
        <Checkbox checked={checkout.acceptedTerms} onCheckedChange={checkout.setAcceptedTerms} />
        <span>I agree to the <Link to="/page/shipping-returns" className="text-primary underline">terms and conditions</Link></span>
      </label>
      {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
      <Button disabled={!checkout.acceptedTerms || placing} onClick={onPlaceOrder}>{placing ? 'Placing order…' : 'Place order'}</Button>
    </div>
  )
}
