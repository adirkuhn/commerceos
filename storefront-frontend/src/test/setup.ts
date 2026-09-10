import '@testing-library/jest-dom/vitest'
import { afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'

afterEach(() => cleanup())

// jsdom doesn't implement matchMedia -- default to "not mobile" so any
// component using it (see components/mobileLayout.ts) renders without
// every test needing its own stub; tests exercising the mobile path
// override this with their own vi.stubGlobal('matchMedia', ...).
if (typeof window !== 'undefined' && !window.matchMedia) {
  window.matchMedia = (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })
}

// jsdom doesn't implement ResizeObserver -- @puckeditor/core's drag/drop
// dependency (@dnd-kit/dom) constructs one at module load, so anything
// that even imports from @puckeditor/core (e.g. CanvasPage.tsx, for its
// editor_mode:'puck' branch) needs this defined before that import runs,
// regardless of whether the test actually exercises Puck rendering.
if (typeof window !== 'undefined' && !window.ResizeObserver) {
  window.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}
