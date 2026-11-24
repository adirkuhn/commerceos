import clsx from 'clsx'
import { paymentStatusLabel, paymentStatusTone, type PaymentTone } from './paymentStatus'
import type { PaymentStatus } from './types'

const TONE_CLASSES: Record<PaymentTone, string> = {
  success: 'bg-success/10 text-success',
  warning: 'bg-warning/10 text-warning',
  danger: 'bg-danger/10 text-danger',
  neutral: 'bg-fill text-ink-4',
}

export function PaymentStatusBadge({ status }: { status: PaymentStatus }) {
  return (
    <span className={clsx('px-2 py-0.5 rounded-sm text-2xs font-semibold', TONE_CLASSES[paymentStatusTone(status)])}>
      {paymentStatusLabel(status)}
    </span>
  )
}
