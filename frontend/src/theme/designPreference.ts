import type { ThemeStorage } from './themePreference'

// Which visual design the admin renders. "depot" is the 2026-10 redesign
// (depot sidebar, Atkinson Hyperlegible type, Today dashboard, reorganised navigation);
// "classic" is the design before it, kept intact so either can be switched
// to at any time -- per person from the account menu, or for everyone by
// building with VITE_DESIGN_DEFAULT=classic.
export type Design = 'classic' | 'depot'

// Also read by the inline script in index.html, which applies the design
// before first paint -- keep the key and class in sync.
export const DESIGN_STORAGE_KEY = 'commerceos.design'
export const DESIGN_CLASS = 'design-depot'

const DESIGNS: readonly Design[] = ['classic', 'depot']

function isDesign(value: unknown): value is Design {
  return typeof value === 'string' && (DESIGNS as readonly string[]).includes(value)
}

// The build-time default. Anything but an explicit "classic" means depot,
// including an unset variable (Vite leaves %VITE_X% literal in index.html).
export function defaultDesign(buildValue: string | undefined): Design {
  return buildValue === 'classic' ? 'classic' : 'depot'
}

export function loadDesignPreference(storage: ThemeStorage | null, fallback: Design): Design {
  try {
    const stored = storage?.getItem(DESIGN_STORAGE_KEY)
    return isDesign(stored) ? stored : fallback
  } catch {
    return fallback
  }
}

export function saveDesignPreference(storage: ThemeStorage | null, design: Design): void {
  try {
    storage?.setItem(DESIGN_STORAGE_KEY, design)
  } catch {
    // Not persisted this time; the in-memory choice still applies.
  }
}
