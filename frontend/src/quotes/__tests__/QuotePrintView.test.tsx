// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QuotePrintView } from '../QuotePrintView'

const lines = [
  { sku: 'FEI-1', product: 'Feijão', quantity: 100, unitPriceMinor: 871 },
  { sku: 'FAR-1', product: 'Farofa', quantity: 5, unitPriceMinor: 1250 },
]

describe('QuotePrintView', () => {
  it('prints sku, product, quantity, price, line totals and the order total only', () => {
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})
    render(<QuotePrintView reference="Q-0007" name="Restaurant Sabor" currency="EUR" lines={lines} onClose={vi.fn()} />)

    expect(print).toHaveBeenCalled()
    expect(screen.getByRole('heading')).toHaveTextContent('Quote Q-0007 — Restaurant Sabor')
    const rows = screen.getAllByRole('row').map(r => [...r.querySelectorAll('th, td')].map(c => c.textContent))
    expect(rows).toEqual([
      ['SKU', 'Product', 'Qty', 'Unit price', 'Line total'],
      ['FEI-1', 'Feijão', '100', '€8.71', '€871.00'],
      ['FAR-1', 'Farofa', '5', '€12.50', '€62.50'],
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
