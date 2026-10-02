import clsx from 'clsx'
import { paymentStatusLabel, paymentStatusTone, type PaymentTone } from './paymentStatus'
import type { PaymentStatus } from './types'

const TONE_CLASSES: Record<PaymentTone, string> = {
  success: 'bg-green-500/15 text-success',
  warning: 'bg-amber-500/15 text-warning',
  danger: 'bg-red-500/15 text-danger',
  neutral: 'bg-gray-500/15 text-ink-4',
}

export function PaymentStatusBadge({ status }: { status: PaymentStatus }) {
  return (
    <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium', TONE_CLASSES[paymentStatusTone(status)])}>
      {paymentStatusLabel(status)}
    </span>
  )
}
