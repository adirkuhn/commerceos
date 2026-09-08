import type { Rect } from './types'

// Fixed mobile design width -- ThemePage.CanvasWidth only ever describes
// the Desktop breakpoint (see domain/theme.Rect's doc comment), so Mobile
// needs its own constant rather than a per-page field. Chosen to match a
// common small-phone viewport (iPhone SE-class); every block is still
// freely resizable within it.
export const MOBILE_CANVAS_WIDTH = 390

// A block with no saved Mobile override (see BlockLayout.Mobile) previews
// full-width at its Desktop y position -- a reasonable "just stack it"
// starting point a merchant can then drag/resize from, not a final
// answer. Kept a pure function of one Rect (not the whole block list) so
// it can't drift from what storefront-frontend computes for the same
// case -- deliberately duplicated there rather than shared across the two
// separate frontend apps/packages.
export function defaultMobileRect(desktop: Rect): Rect {
  return { x: 0, y: desktop.y, width: MOBILE_CANVAS_WIDTH, height: desktop.height }
}
