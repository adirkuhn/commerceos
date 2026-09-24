// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { AddItemPanel } from '../AddItemPanel'

function renderPanel(resolveCode = vi.fn().mockResolvedValue({ sku_id: 'sku-9', product_name: 'Jam jar' })) {
  const onAdd = vi.fn()
  const onClose = vi.fn()
  render(<AddItemPanel busy={false} resolveCode={resolveCode} onAdd={onAdd} onClose={onClose} />)
  return { onAdd, onClose, resolveCode }
}

describe('AddItemPanel', () => {
  it('scans the product, then adds it as a freebie by default', async () => {
    const user = userEvent.setup()
    const { onAdd, resolveCode } = renderPanel()

    await user.type(screen.getByPlaceholderText('Scan or type SKU/barcode'), 'JAM-1{Enter}')
    expect(resolveCode).toHaveBeenCalledWith('JAM-1')
    expect(await screen.findByText('Jam jar')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Add to order' }))

    expect(onAdd).toHaveBeenCalledWith({ skuId: 'sku-9', name: 'Jam jar', freebie: true, quantity: 1, priceMinorUnits: undefined })
  })

  it('a paid new item carries its price in minor units', async () => {
    const user = userEvent.setup()
    const { onAdd } = renderPanel()
    await user.type(screen.getByPlaceholderText('Scan or type SKU/barcode'), 'JAM-1{Enter}')
    await screen.findByText('Jam jar')

    await user.click(screen.getByRole('button', { name: /New item/ }))
    await user.clear(screen.getByLabelText('Quantity'))
    await user.type(screen.getByLabelText('Quantity'), '2')
    await user.clear(screen.getByLabelText('Price'))
    await user.type(screen.getByLabelText('Price'), '3.99')
    await user.click(screen.getByRole('button', { name: 'Add to order' }))

    expect(onAdd).toHaveBeenCalledWith({ skuId: 'sku-9', name: 'Jam jar', freebie: false, quantity: 2, priceMinorUnits: 399 })
  })

  it('shows an unrecognized code', async () => {
    const user = userEvent.setup()
    renderPanel(vi.fn().mockRejectedValue(new Error('sku not found')))

    await user.type(screen.getByPlaceholderText('Scan or type SKU/barcode'), 'NOPE{Enter}')

    expect(await screen.findByRole('alert')).toHaveTextContent('sku not found')
  })

  it('freebie/new item buttons expose which one is active', async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.type(screen.getByPlaceholderText('Scan or type SKU/barcode'), 'JAM-1{Enter}')
    await screen.findByText('Jam jar')

    expect(screen.getByRole('button', { name: /Freebie/ })).toHaveAttribute('aria-pressed', 'true')
    await user.click(screen.getByRole('button', { name: /New item/ }))
    expect(screen.getByRole('button', { name: /New item/ })).toHaveAttribute('aria-pressed', 'true')
  })
})
