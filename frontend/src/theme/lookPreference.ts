import type { ThemeStorage } from './themePreference'

// Which look (palette, type and material) the admin wears. Both looks share
// one layout and one set of components; each has a light and a dark
// appearance (see themePreference). Chosen per browser from the account menu.
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

export function loadLookPreference(storage: ThemeStorage | null): Look {
  try {
    const stored = storage?.getItem(LOOK_STORAGE_KEY)
    return isLook(stored) ? stored : DEFAULT_LOOK
  } catch {
    return DEFAULT_LOOK
  }
}

export function saveLookPreference(storage: ThemeStorage | null, look: Look): void {
  try {
    storage?.setItem(LOOK_STORAGE_KEY, look)
  } catch {
    // Not persisted this time; the in-memory choice still applies.
  }
}
