import { describe, it, expect } from 'vitest'
import { defaultMobileRect, MOBILE_CANVAS_WIDTH } from '../mobileLayout'

describe('defaultMobileRect', () => {
  it('goes full mobile width, keeps the desktop y and height', () => {
    const result = defaultMobileRect({ x: 100, y: 250, width: 600, height: 300 })

    expect(result).toEqual({ x: 0, y: 250, width: MOBILE_CANVAS_WIDTH, height: 300 })
  })
})
