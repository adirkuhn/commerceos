// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { LineChangeNotes, NotAddedExtras } from '../LineChangeNotes'
import { change } from './fixtures'

describe('LineChangeNotes', () => {
  it('shows which product replaced the line, how much it covers and why', () => {
    render(<LineChangeNotes linePriceMinor={249} currency="EUR" changes={[
      change({ product_name: 'Feijão Preto 1kg', variant_name: 'Feijão Preto 1kg', sku_code: 'FEP-1', quantity: 4,
        original_qty_covered: 4, reason: 'out of stock', return_to_stock: true }),
    ]} />)

    expect(screen.getByText(/Replaced by Feijão Preto 1kg × 4/)).toHaveTextContent('Replaced by Feijão Preto 1kg × 4 (FEP-1) · covers 4 of this line')
    expect(screen.getByText(/out of stock/)).toHaveTextContent(/original back to stock/)
  })

  it('shows a refund with its amount at the line price', () => {
    render(<LineChangeNotes linePriceMinor={199} currency="EUR" changes={[change({ action: 'refund', quantity: 2, reason: 'damaged' })]} />)

    expect(screen.getByText(/Refunded 2/)).toHaveTextContent(/3[.,]98/)
  })

  it('marks free and charged extra items', () => {
    render(<LineChangeNotes linePriceMinor={0} currency="EUR" changes={[
      change({ id: 'a', action: 'freebie', reason: 'Guaraná' }),
      change({ id: 'b', action: 'new_item', reason: 'Guaraná', price_minor_units: 150 }),
    ]} />)

    expect(screen.getByText('Extra · free')).toBeInTheDocument()
    expect(screen.getByText('Extra · charged')).toBeInTheDocument()
    expect(screen.queryByText(/“Guaraná”/)).not.toBeInTheDocument()
  })

  it('says when the change was made at picking', () => {
    render(<LineChangeNotes linePriceMinor={0} currency="EUR" changes={[change({ step: 'picking' })]} />)

    expect(screen.getByText(/at picking/)).toBeInTheDocument()
  })

  it('renders nothing for an untouched line', () => {
    const { container } = render(<LineChangeNotes linePriceMinor={0} currency="EUR" changes={[]} />)
    expect(container).toBeEmptyDOMElement()
  })
})

describe('NotAddedExtras', () => {
  it('lists extras shipped without', () => {
    render(<NotAddedExtras changes={[change({ action: 'freebie', line_item_id: null, product_name: 'Guaraná', variant_name: '', quantity: 1 })]} />)

    expect(screen.getByText('Not added (no stock): Guaraná × 1')).toBeInTheDocument()
  })
})
