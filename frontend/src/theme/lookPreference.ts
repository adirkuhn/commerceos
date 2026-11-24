import type { ThemeStorage } from './themePreference'

// Which look (palette, type and material) the admin wears. Both looks share
// one layout and one set of components; each has a light and a dark
// appearance (see themePreference). The shop sets a default (Settings >
// Appearance); a person may pick their own per browser from the account
// menu, which wins until they go back to the shop default.
//   wallet   -- Ticket Wallet: carrier navy, coupon white, carbon-purple figures
//   graphite -- Graphite Console: graphite panels split by hairline seams
export type Look = 'wallet' | 'graphite'

export const LOOKS: readonly Look[] = ['wallet', 'graphite']
export const DEFAULT_LOOK: Look = 'wallet'

// Also read by the inline script in index.html, which applies the look
// before first paint -- keep the key and classes in sync.
export const LOOK_STORAGE_KEY = 'commerceos.look'

export function lookClass(look: Look): string {
  return `look-${look}`
}

function isLook(value: unknown): value is Look {
  return typeof value === 'string' && (LOOKS as readonly string[]).includes(value)
}

// A storage that may also forget a key (localStorage does).
export type LookStorage = ThemeStorage & { removeItem?: (key: string) => void }

// This browser's own choice, or null to follow the shop default.
export function loadLookPreference(storage: LookStorage | null): Look | null {
  try {
    const stored = storage?.getItem(LOOK_STORAGE_KEY)
    return isLook(stored) ? stored : null
  } catch {
    return null
  }
}

export function saveLookPreference(storage: LookStorage | null, look: Look | null): void {
  try {
    if (look) storage?.setItem(LOOK_STORAGE_KEY, look)
    else storage?.removeItem?.(LOOK_STORAGE_KEY)
  } catch {
    // Not persisted this time; the in-memory choice still applies.
  }
}

// The look to show: the person's, else the shop's, else Ticket Wallet.
export function resolveLook(personal: Look | null, shopDefault: Look | null | undefined): Look {
  return personal ?? shopDefault ?? DEFAULT_LOOK
}
