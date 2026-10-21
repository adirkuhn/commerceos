import type { TFunction } from 'i18next'
import { providerLabel } from '../webhooks/providerLabel'
import { MISSING_SETUP, type MissingSetup, type ShippingConnection } from './types'

// What each missing setup step tells staff -- the one place that maps a
// backend `missing` reason to text.
const MISSING_SETUP_KEYS = {
  [MISSING_SETUP.senderAddress]: 'readiness.senderAddress',
} as const satisfies Record<MissingSetup, string>

// Whether the connection can create labels right now.
export function connectionReady(connection: Pick<ShippingConnection, 'missing'> | null | undefined): boolean {
  return !connection || connection.missing.length === 0
}

// One sentence per missing step, naming the connection's provider.
export function missingSetupTexts(t: TFunction<'shipping'>, connection: Pick<ShippingConnection, 'provider' | 'missing'>): string[] {
  const provider = providerLabel(connection.provider)
  return connection.missing.map(reason => t(MISSING_SETUP_KEYS[reason], { provider }))
}
