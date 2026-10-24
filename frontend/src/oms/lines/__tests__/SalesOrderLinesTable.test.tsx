// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { SalesOrderLinesTable } from '../SalesOrderLinesTable'
import type { SalesOrderLineItem } from '../../types'
import { change } from '../../lineChanges/__tests__/fixtures'

const woo: SalesOrderLineItem = {
  id: 'li-caldo', sku_id: 'sku-1', name: 'Caldo de Cana', sku_code: 'CAL-1', barcode: '', quantity: 3, price_minor_units: 458,
  subtotal_minor_units: 1373, subtotal_tax_minor_units: 124, total_minor_units: 1373, total_tax_minor_units: 124,
  total_excl_tax_minor_units: 1373, total_incl_tax_minor_units: 1497,
}

describe('SalesOrderLinesTable', () => {
  it('shows each line without VAT, its VAT and with VAT', () => {
    render(<SalesOrderLinesTable lines={[woo]} changesByLine={new Map()} currency="EUR" />)

    for (const header of ['Excl. VAT', 'VAT', 'Incl. VAT']) expect(screen.getByRole('columnheader', { name: header })).toBeInTheDocument()
    const row = screen.getByRole('row', { name: /Caldo de Cana/ })
    const cells = within(row).getAllByRole('cell')
    expect(cells[1]).toHaveTextContent(/^3/)
    expect(cells[1]).toHaveTextContent(/4[.,]99 each/)
    expect(cells[2]).toHaveTextContent(/13[.,]73/)
    expect(cells[3]).toHaveTextContent(/1[.,]24/)
    expect(cells[4]).toHaveTextContent(/14[.,]97/)
  })

  it('shows what picking and packing changed under the line', () => {
    render(<SalesOrderLinesTable lines={[woo]} currency="EUR"
      changesByLine={new Map([['li-caldo', [change({ action: 'refund', quantity: 3, amount_minor_units: 1497 })]]])} />)

    expect(screen.getByText(/Refunded 3/)).toHaveTextContent(/14[.,]97/)
  })

  it('flags a line not linked to a product', () => {
    render(<SalesOrderLinesTable lines={[{ ...woo, sku_id: null }]} changesByLine={new Map()} currency="EUR" />)

    expect(screen.getByText('Not linked to a product yet')).toBeInTheDocument()
  })
})
