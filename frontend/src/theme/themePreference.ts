// Appearance preference. "system" follows the OS light/dark setting (the
// default -- people expect an app to match the rest of their desktop);
// light/dark are explicit overrides the Topbar can cycle through.
export type ThemePreference = 'system' | 'light' | 'dark'
export type ResolvedTheme = 'light' | 'dark'

// Also read by the inline script in index.html, which applies the theme
// before first paint -- keep the two in sync.
export const THEME_STORAGE_KEY = 'commerceos.theme'

const PREFERENCES: readonly ThemePreference[] = ['system', 'light', 'dark']

// The subset of Storage this module needs, so tests (and a blocked
// localStorage) can supply their own.
export interface ThemeStorage {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
}

function isThemePreference(value: unknown): value is ThemePreference {
  return typeof value === 'string' && (PREFERENCES as readonly string[]).includes(value)
}

export function resolveTheme(preference: ThemePreference, systemPrefersDark: boolean): ResolvedTheme {
  if (preference === 'system') return systemPrefersDark ? 'dark' : 'light'
  return preference
}

export function nextThemePreference(preference: ThemePreference): ThemePreference {
  return PREFERENCES[(PREFERENCES.indexOf(preference) + 1) % PREFERENCES.length]
}

// Storage can throw (private windows, blocked site data) -- the preference
// is a convenience, so any failure just means "system".
export function loadThemePreference(storage: ThemeStorage | null): ThemePreference {
  try {
    const stored = storage?.getItem(THEME_STORAGE_KEY)
    return isThemePreference(stored) ? stored : 'system'
  } catch {
    return 'system'
  }
}

export function saveThemePreference(storage: ThemeStorage | null, preference: ThemePreference): void {
  try {
    storage?.setItem(THEME_STORAGE_KEY, preference)
  } catch {
    // Not persisted this time; the in-memory preference still applies.
  }
}
