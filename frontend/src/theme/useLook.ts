import { useCallback, useReducer, useSyncExternalStore } from 'react'
import {
  DEFAULT_LOOK,
  LOOKS,
  loadLookPreference,
  lookClass,
  resolveLook,
  saveLookPreference,
  type Look,
  type LookStorage,
} from './lookPreference'

// The look class on <html> is the single source of truth for what shows:
// the pre-paint script in index.html sets it, initLook re-applies it at
// boot, setShopDefaultLook follows the shop's setting and setLook records a
// person's own choice. Readers subscribe to the class, so any component can
// read the look without app context.

function browserStorage(): LookStorage | null {
  try {
    return window.localStorage
  } catch {
    return null
  }
}

// The shop's default look, once known (AppearanceProvider sets it).
let shopDefault: Look | null = null

export function applyLook(look: Look): void {
  const classes = document.documentElement.classList
  for (const other of LOOKS) classes.toggle(lookClass(other), other === look)
}

function applyResolved(): void {
  applyLook(resolveLook(loadLookPreference(browserStorage()), shopDefault))
}

export function initLook(): void {
  applyResolved()
}

// The shop's setting changed or loaded: it shows unless the person picked
// their own look. null forgets it (tests, signing out).
export function setShopDefaultLook(look: Look | null): void {
  shopDefault = look
  applyResolved()
}

function subscribe(onChange: () => void): () => void {
  const observer = new MutationObserver(onChange)
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  return () => observer.disconnect()
}

function snapshot(): Look {
  const classes = document.documentElement.classList
  return LOOKS.find(look => classes.contains(lookClass(look))) ?? DEFAULT_LOOK
}

export interface LookState {
  // What shows now.
  look: Look
  // This browser's own choice, or null when it follows the shop default.
  personal: Look | null
  // A look of one's own, or null to follow the shop default again.
  setLook: (look: Look | null) => void
}

export function useLook(): LookState {
  const look = useSyncExternalStore(subscribe, snapshot, () => DEFAULT_LOOK)
  // Switching between "my own" and "the shop's" can leave the class as it
  // was (same look) -- re-render so `personal` is read fresh.
  const [, rerender] = useReducer((n: number) => n + 1, 0)
  const setLook = useCallback((next: Look | null) => {
    saveLookPreference(browserStorage(), next)
    applyResolved()
    rerender()
  }, [])
  return { look, personal: loadLookPreference(browserStorage()), setLook }
}
