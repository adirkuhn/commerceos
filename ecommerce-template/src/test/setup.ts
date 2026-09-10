import '@testing-library/jest-dom/vitest'

// jsdom has no PointerEvent constructor at all -- Radix UI primitives
// (Dialog/DropdownMenu/Tooltip/Drawer...) open on pointerdown and read
// pointer-specific fields (pointerType, pointerId) off the event, and
// userEvent's realistic pointer sequence hangs indefinitely without a
// real PointerEvent to dispatch. This is the standard minimal polyfill
// (mirrors what jsdom-testing-mocks and similar packages ship).
if (typeof window !== 'undefined' && !window.PointerEvent) {
  class PointerEventPolyfill extends MouseEvent {
    pointerId: number
    pointerType: string
    isPrimary: boolean
    width: number
    height: number
    pressure: number
    tangentialPressure: number
    tiltX: number
    tiltY: number
    twist: number

    constructor(type: string, params: PointerEventInit = {}) {
      super(type, params)
      this.pointerId = params.pointerId ?? 0
      this.pointerType = params.pointerType ?? 'mouse'
      this.isPrimary = params.isPrimary ?? true
      this.width = params.width ?? 1
      this.height = params.height ?? 1
      this.pressure = params.pressure ?? 0
      this.tangentialPressure = params.tangentialPressure ?? 0
      this.tiltX = params.tiltX ?? 0
      this.tiltY = params.tiltY ?? 0
      this.twist = params.twist ?? 0
    }
  }
  // @ts-expect-error -- assigning a same-shape polyfill, not jsdom's (absent) type
  window.PointerEvent = PointerEventPolyfill
}

// Radix UI primitives also call these pointer-capture/scroll APIs on
// open -- jsdom doesn't implement them, and without a stub, interacting
// with a Radix trigger throws instead of opening.
if (!Element.prototype.hasPointerCapture) {
  Element.prototype.hasPointerCapture = () => false
}
if (!Element.prototype.setPointerCapture) {
  Element.prototype.setPointerCapture = () => {}
}
if (!Element.prototype.releasePointerCapture) {
  Element.prototype.releasePointerCapture = () => {}
}
if (!Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = () => {}
}
