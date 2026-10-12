import { useCallback, useSyncExternalStore } from 'react'
import {
  DESIGN_CLASS,
  loadDesignPreference,
  saveDesignPreference,
  type Design,
} from './designPreference'
import type { ThemeStorage } from './themePreference'

// The class on <html> is the single source of truth for the design: the
// pre-paint script in index.html sets it, initDesign re-applies it at boot
// and setDesign changes it. Readers subscribe to the class, so any
// component can branch on the design without app context.

function browserStorage(): ThemeStorage | null {
  try {
    return window.localStorage
  } catch {
    return null
  }
}

export function applyDesign(design: Design): void {
  document.documentElement.classList.toggle(DESIGN_CLASS, design === 'depot')
}

export function initDesign(fallback: Design): void {
  applyDesign(loadDesignPreference(browserStorage(), fallback))
}

function subscribe(onChange: () => void): () => void {
  const observer = new MutationObserver(onChange)
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  return () => observer.disconnect()
}

function snapshot(): Design {
  return document.documentElement.classList.contains(DESIGN_CLASS) ? 'depot' : 'classic'
}

export interface DesignState {
  design: Design
  setDesign: (design: Design) => void
}

export function useDesign(): DesignState {
  const design = useSyncExternalStore(subscribe, snapshot, () => 'classic' as Design)
  const setDesign = useCallback((next: Design) => {
    saveDesignPreference(browserStorage(), next)
    applyDesign(next)
  }, [])
  return { design, setDesign }
}
