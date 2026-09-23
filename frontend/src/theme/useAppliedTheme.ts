import { useSyncExternalStore } from 'react'
import type { ResolvedTheme } from './themePreference'

// The appearance actually applied to the page (<html class="dark">, set by
// useTheme or the pre-paint script). For code that needs concrete colours
// -- e.g. charts, whose SVG attributes can't read CSS variables -- without
// depending on AppContext.
function subscribe(onChange: () => void): () => void {
  const observer = new MutationObserver(onChange)
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  return () => observer.disconnect()
}

function snapshot(): ResolvedTheme {
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

export function useAppliedTheme(): ResolvedTheme {
  return useSyncExternalStore(subscribe, snapshot, () => 'light')
}
