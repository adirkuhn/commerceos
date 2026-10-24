// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QuotePrintView } from '../QuotePrintView'

const lines = [
  { product: 'Feijão — 1kg (FEI-1)', quantity: 100, unitPrice: 871 },
  { product: 'Farofa (FAR-1)', quantity: 5, unitPrice: 1250 },
]

describe('QuotePrintView', () => {
  it('prints product, quantity, price, line totals and the order total only', () => {
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})
    render(<QuotePrintView reference="Q-0007" name="Restaurant Sabor" currency="EUR" lines={lines} onClose={vi.fn()} />)

    expect(print).toHaveBeenCalled()
    expect(screen.getByRole('heading')).toHaveTextContent('Quote Q-0007 — Restaurant Sabor')
    const rows = screen.getAllByRole('row').map(r => [...r.querySelectorAll('th, td')].map(c => c.textContent))
    expect(rows).toEqual([
      ['Product', 'Qty', 'Unit price', 'Line total'],
      ['Feijão — 1kg (FEI-1)', '100', '€8.71', '€871.00'],
      ['Farofa (FAR-1)', '5', '€12.50', '€62.50'],
      ['Order total', '€933.50'],
    ])
    print.mockRestore()
  })

  it('an unsaved quote has no number', () => {
    vi.spyOn(window, 'print').mockImplementation(() => {})
    render(<QuotePrintView reference={null} name="" currency="EUR" lines={lines} onClose={vi.fn()} />)
    expect(screen.getByRole('heading')).toHaveTextContent(/^Quote$/)
  })

  it('closes', () => {
    vi.spyOn(window, 'print').mockImplementation(() => {})
    const onClose = vi.fn()
    render(<QuotePrintView reference={null} name="x" currency="EUR" lines={lines} onClose={onClose} />)
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(onClose).toHaveBeenCalled()
  })
})
