import { CheckCircle, XCircle } from 'lucide-react'
import { isTenantAdmin } from '../auth/roles'

// Same tier as WooCommerceSyncSettingsPage's webhook-secret tab -- this
// table exposes raw (if opaque) third-party payloads and a manual
// republish action, security/ops-adjacent rather than day-to-day
// integration use, see that file's canManageWebhooks doc comment.
export function canManageWebhooks(role: string | undefined): boolean {
  return isTenantAdmin(role)
}

export const PROVIDER_COLORS: Record<string, string> = {
  woocommerce: 'from-purple-500 to-violet-600',
  bol: 'from-blue-500 to-cyan-600',
  myparcel: 'from-orange-500 to-red-600',
  postnl: 'from-orange-400 to-amber-500',
  moneybird: 'from-sky-500 to-blue-600',
  whatsapp: 'from-green-500 to-emerald-600',
  stripe: 'from-indigo-500 to-purple-600',
  revolut: 'from-slate-700 to-black',
}

export const CURRENCIES = ['USD', 'EUR', 'GBP', 'CAD', 'AUD']

// Currency codes saved before the backend started normalizing them
// (uppercase, trimmed) won't exact-match a <select> option otherwise —
// normalize on read too so existing connections aren't stuck unselected.
export function normalizeCurrency(value: string | null | undefined): string {
  return (value ?? '').trim().toUpperCase()
}

export const STATUS_ICON = {
  connected: CheckCircle,
  error: XCircle,
}

export const STATUS_COLOR = {
  connected: 'text-success',
  error: 'text-danger',
}
