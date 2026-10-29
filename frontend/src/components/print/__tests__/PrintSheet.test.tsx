// @vitest-environment jsdom
import { render } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { PrintSheet } from '../PrintSheet'

describe('PrintSheet', () => {
  it('renders straight under <body>, outside the app shell that clips at screen height', () => {
    const { getByText } = render(
      <div id="app-shell" className="h-screen overflow-hidden"><PrintSheet><p>Invoice body</p></PrintSheet></div>,
    )

    const sheet = getByText('Invoice body').closest('.print-sheet')
    expect(sheet).not.toBeNull()
    expect(sheet?.parentElement).toBe(document.body)
    expect(document.getElementById('app-shell')?.contains(sheet ?? null)).toBe(false)
  })

  it('marks <body> while open so print CSS shows only the sheet, and unmarks it on close', () => {
    const { unmount } = render(<PrintSheet><p>x</p></PrintSheet>)
    expect(document.body.classList.contains('printing')).toBe(true)

    unmount()
    expect(document.body.classList.contains('printing')).toBe(false)
  })
})
