// @vitest-environment jsdom
import { render } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { BrandMark } from '../BrandMark'

describe('BrandMark', () => {
  it('is decorative: hidden from assistive tech, never a tab stop', () => {
    const { container } = render(<BrandMark />)
    const svg = container.querySelector('svg')!
    expect(svg).toHaveAttribute('aria-hidden', 'true')
    expect(svg).toHaveAttribute('focusable', 'false')
  })

  it('draws in currentColor so it follows the surrounding text colour', () => {
    const { container } = render(<BrandMark />)
    expect(container.innerHTML).toContain('currentColor')
    expect(container.innerHTML).not.toMatch(/#[0-9a-f]{3,6}/i)
  })

  it('scales to the requested size', () => {
    const { container } = render(<BrandMark size={32} />)
    const svg = container.querySelector('svg')!
    expect(svg).toHaveAttribute('width', '32')
    expect(svg).toHaveAttribute('height', '32')
  })
})
