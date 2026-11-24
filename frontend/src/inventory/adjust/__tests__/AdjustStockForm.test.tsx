// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { AdjustStockForm } from '../AdjustStockForm'
import { stockUnitFor } from '../../stockUnit'
import type { StockSlot } from '../stockSlots'

const slot = (over: Partial<StockSlot>): StockSlot => ({
  key: 'w1:', warehouseId: 'w1', warehouseName: 'Main', batchId: '', lot: '', expiryDate: null,
  quantity: 0, isLedger: true, empty: false, ...over,
})
const slots: StockSlot[] = [
  slot({ quantity: 4 }),
  slot({ key: 'w1:b1', batchId: 'b1', lot: 'L-2411', quantity: 12, isLedger: false }),
  slot({ key: 'w1:b0', batchId: 'b0', lot: 'L-OLD', quantity: 0, isLedger: false, empty: true }),
]

function setup(props: Partial<Parameters<typeof AdjustStockForm>[0]> = {}) {
  const onSave = vi.fn().mockResolvedValue(undefined)
  const onRecorded = vi.fn()
  render(
    <AdjustStockForm slots={slots} unit={stockUnitFor()} formatQuantity={String} onSave={onSave} onRecorded={onRecorded} {...props} />,
  )
  return { onSave, onRecorded, user: userEvent.setup() }
}

describe('AdjustStockForm', () => {
  it('adds received stock to the picked lot and previews before → after', async () => {
    const { onSave, onRecorded, user } = setup()
    await user.click(screen.getByRole('radio', { name: /L-2411/ }))
    await user.type(screen.getByLabelText('Quantity'), '3')
    expect(screen.getByTestId('adjust-preview')).toHaveTextContent('12 → 15')
    await user.click(screen.getByRole('button', { name: 'Record change' }))
    expect(onSave).toHaveBeenCalledWith({ warehouse_id: 'w1', batch_id: 'b1', type: 'receipt', quantity_delta: 3, reason: 'Received' })
    expect(onRecorded).toHaveBeenCalledWith(expect.objectContaining({ before: 12, after: 15 }))
  })

  it('removing switches to an outgoing reason and records a negative sale with the note', async () => {
    const { onSave, user } = setup({ initialKey: 'w1:b1' })
    await user.click(screen.getByRole('button', { name: 'Remove' }))
    expect(screen.getByRole('button', { name: 'Sold' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.queryByRole('button', { name: 'Received' })).toBeNull()
    await user.type(screen.getByLabelText('Quantity'), '2')
    await user.type(screen.getByLabelText('Note (optional)'), 'market')
    await user.click(screen.getByRole('button', { name: 'Record change' }))
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ type: 'sale', quantity_delta: -2, reason: 'Sold — market' }))
  })

  it('will not remove more than is on hand', async () => {
    const { onSave, user } = setup({ initialKey: 'w1:' })
    await user.click(screen.getByRole('button', { name: 'Remove' }))
    await user.type(screen.getByLabelText('Quantity'), '9')
    expect(screen.getByText('Only 4 on hand here.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Record change' })).toBeDisabled()
    expect(onSave).not.toHaveBeenCalled()
  })

  it('asks for a quantity before recording', async () => {
    const { onSave, user } = setup()
    await user.click(screen.getByRole('button', { name: 'Record change' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Enter a quantity above zero.')
    expect(onSave).not.toHaveBeenCalled()
  })

  it('shows why the server refused', async () => {
    const { onSave, user } = setup()
    onSave.mockRejectedValueOnce(new Error('boom'))
    await user.type(screen.getByLabelText('Quantity'), '1')
    await user.click(screen.getByRole('button', { name: 'Record change' }))
    expect(await screen.findByRole('alert')).toBeInTheDocument()
  })

  it('hides empty lots until asked', async () => {
    const { user } = setup()
    expect(screen.queryByRole('radio', { name: /L-OLD/ })).toBeNull()
    await user.click(screen.getByRole('button', { name: 'Show 1 empty lot' }))
    expect(screen.getByRole('radio', { name: /L-OLD/ })).toBeInTheDocument()
  })

  it('offers only the locked place when opened from a row', () => {
    setup({ initialKey: 'w1:b1', locked: true })
    expect(screen.queryAllByRole('radio')).toHaveLength(0)
    expect(within(screen.getByTestId('adjust-where')).getByText(/L-2411/)).toBeInTheDocument()
  })

  it('points damaged or lost stock at the write-off', async () => {
    const onWriteOff = vi.fn()
    const { user } = setup({ onWriteOff })
    expect(screen.queryByRole('button', { name: 'Write it off' })).toBeNull()
    await user.click(screen.getByRole('button', { name: 'Remove' }))
    await user.click(screen.getByRole('button', { name: 'Write it off' }))
    expect(onWriteOff).toHaveBeenCalled()
  })
})

describe('AdjustStockForm default place', () => {
  it('follows the places as they load until the person picks one', () => {
    const props = { unit: stockUnitFor(), formatQuantity: String, onSave: vi.fn(), onRecorded: vi.fn() }
    const { rerender } = render(<AdjustStockForm {...props} slots={[slots[0]].map(s => ({ ...s, quantity: 0 }))} />)
    expect(screen.getByRole('radio', { name: /No lot/ })).toBeChecked()
    rerender(<AdjustStockForm {...props} slots={[{ ...slots[0], quantity: 0 }, slots[1]]} />)
    expect(screen.getByRole('radio', { name: /L-2411/ })).toBeChecked()
  })
})
