import { useCallback, useSyncExternalStore } from 'react'
import {
  DEFAULT_LOOK,
  LOOKS,
  loadLookPreference,
  lookClass,
  saveLookPreference,
  type Look,
} from './lookPreference'
import type { ThemeStorage } from './themePreference'

// The look class on <html> is the single source of truth: the pre-paint
// script in index.html sets it, initLook re-applies it at boot and setLook
// changes it. Readers subscribe to the class, so any component can read the
// look without app context.

function browserStorage(): ThemeStorage | null {
  try {
    return window.localStorage
  } catch {
    return null
  }
}

export function applyLook(look: Look): void {
  const classes = document.documentElement.classList
  for (const other of LOOKS) classes.toggle(lookClass(other), other === look)
}

export function initLook(): void {
  applyLook(loadLookPreference(browserStorage()))
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
  look: Look
  setLook: (look: Look) => void
}

export function useLook(): LookState {
  const look = useSyncExternalStore(subscribe, snapshot, () => DEFAULT_LOOK)
  const setLook = useCallback((next: Look) => {
    saveLookPreference(browserStorage(), next)
    applyLook(next)
  }, [])
  return { look, setLook }
}
