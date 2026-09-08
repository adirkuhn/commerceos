import { useEffect, useState } from 'react'
import type { Rect } from '../types'

// Matches the admin editor's own MOBILE_CANVAS_WIDTH/defaultMobileRect
// (frontend/src/theme/mobileLayout.ts) -- deliberately duplicated rather
// than shared across these two separate frontend apps/packages. A real
// viewport narrower than this still renders correctly: CanvasPage wraps
// the canvas in a horizontally-scrolling container.
export const MOBILE_CANVAS_WIDTH = 390

const MOBILE_QUERY = '(max-width: 480px)'

export function useIsMobileViewport(): boolean {
  const [isMobile, setIsMobile] = useState(() => typeof window !== 'undefined' && window.matchMedia(MOBILE_QUERY).matches)

  useEffect(() => {
    const mql = window.matchMedia(MOBILE_QUERY)
    const handler = (e: MediaQueryListEvent) => setIsMobile(e.matches)
    mql.addEventListener('change', handler)
    return () => mql.removeEventListener('change', handler)
  }, [])

  return isMobile
}

// A block with no saved Mobile override falls back to a full-width
// preview at its Desktop y position -- see the admin editor's identical
// defaultMobileRect for the full reasoning.
export function defaultMobileRect(desktop: Rect): Rect {
  return { x: 0, y: desktop.y, width: MOBILE_CANVAS_WIDTH, height: desktop.height }
}
