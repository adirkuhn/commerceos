import { CreditCard, Wallet, Smartphone, Landmark, Banknote, Clock } from 'lucide-react'
import type { PaymentMethodId, PaymentMethodOption } from '@/types'
import { RadioGroup, RadioGroupItem } from '@/components/ui/RadioGroup'

// Every option is a UI-only choice -- no real payment processing per
// spec section 6 ("Do not implement real payment processing"). Selecting
// "card" would, in a real integration, mount that provider's own hosted
// card element here (Stripe Elements, Braintree, etc) behind this same
// interface; a custom provider plugs in the same way via
// EcommercePlugin.checkoutExtensions.
export const PAYMENT_METHODS: PaymentMethodOption[] = [
  { id: 'card', label: 'Credit / debit card', description: 'Visa, Mastercard, Amex', icon: 'card' },
  { id: 'paypal', label: 'PayPal', description: 'Redirect to PayPal to complete payment', icon: 'wallet' },
  { id: 'apple_pay', label: 'Apple Pay', description: 'Pay with Touch ID or Face ID', icon: 'phone' },
  { id: 'google_pay', label: 'Google Pay', description: 'Pay with your Google account', icon: 'phone' },
  { id: 'bank_transfer', label: 'Bank transfer', description: 'Pay by direct bank transfer', icon: 'bank' },
  { id: 'cod', label: 'Cash on delivery', description: 'Pay when your order arrives', icon: 'cash' },
  { id: 'bnpl', label: 'Buy now, pay later', description: 'Split into 4 interest-free payments', icon: 'clock' },
]

const ICONS: Record<string, typeof CreditCard> = { card: CreditCard, wallet: Wallet, phone: Smartphone, bank: Landmark, cash: Banknote, clock: Clock }

export function PaymentMethodSelector({ value, onChange }: { value: PaymentMethodId; onChange: (id: PaymentMethodId) => void }) {
  return (
    <RadioGroup value={value} onValueChange={(v) => onChange(v as PaymentMethodId)} name="payment-method">
      {PAYMENT_METHODS.map(method => {
        const Icon = ICONS[method.icon] ?? CreditCard
        return (
          <label
            key={method.id}
            htmlFor={`payment-${method.id}`}
            className="flex cursor-pointer items-center gap-3 rounded-[var(--radius)] border border-border p-4 has-[:checked]:border-primary has-[:checked]:bg-accent"
          >
            <RadioGroupItem id={`payment-${method.id}`} value={method.id} />
            <Icon className="h-5 w-5 text-muted-foreground" />
            <div>
              <p className="text-sm font-medium">{method.label}</p>
              <p className="text-xs text-muted-foreground">{method.description}</p>
            </div>
          </label>
        )
      })}
    </RadioGroup>
  )
}
