// @vitest-environment jsdom
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { StockShortfallDialog } from '../StockShortfallDialog'
import type { StockShortfall } from '../types'

function shortfall(overrides: Partial<StockShortfall> = {}): StockShortfall {
  return { scan_id: 'scan-1', sku_id: 'sku-1', name: 'Widget', needed: 5, short: 3, ...overrides }
}

describe('StockShortfallDialog', () => {
  it('defaults every item to "with_item" and confirms with that choice', () => {
    const onConfirm = vi.fn()
    render(<StockShortfallDialog shortfalls={[shortfall()]} submitting={false} onConfirm={onConfirm} onCancel={vi.fn()} />)

    expect(screen.getByText('Widget')).toBeInTheDocument()
    expect(screen.getByText('Needed 5, 3 short')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Confirm & Complete' }))
    expect(onConfirm).toHaveBeenCalledWith({ 'scan-1': 'with_item' })
  })

  it('lets staff switch an item to "without_item" before confirming', () => {
    const onConfirm = vi.fn()
    render(<StockShortfallDialog shortfalls={[shortfall()]} submitting={false} onConfirm={onConfirm} onCancel={vi.fn()} />)

    fireEvent.click(screen.getByRole('button', { name: 'Skip it' }))
    fireEvent.click(screen.getByRole('button', { name: 'Confirm & Complete' }))

    expect(onConfirm).toHaveBeenCalledWith({ 'scan-1': 'without_item' })
  })

  it('tracks each item independently', () => {
    const onConfirm = vi.fn()
    render(
      <StockShortfallDialog
        shortfalls={[shortfall({ scan_id: 'scan-1', name: 'Widget' }), shortfall({ scan_id: 'scan-2', name: 'Gadget' })]}
        submitting={false} onConfirm={onConfirm} onCancel={vi.fn()}
      />,
    )

    const skipButtons = screen.getAllByRole('button', { name: 'Skip it' })
    fireEvent.click(skipButtons[0])
    fireEvent.click(screen.getByRole('button', { name: 'Confirm & Complete' }))

    expect(onConfirm).toHaveBeenCalledWith({ 'scan-1': 'without_item', 'scan-2': 'with_item' })
  })

  it('calls onCancel when Cancel is clicked', () => {
    const onCancel = vi.fn()
    render(<StockShortfallDialog shortfalls={[shortfall()]} submitting={false} onConfirm={vi.fn()} onCancel={onCancel} />)

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onCancel).toHaveBeenCalled()
  })

  it('disables actions while submitting', () => {
    render(<StockShortfallDialog shortfalls={[shortfall()]} submitting={true} onConfirm={vi.fn()} onCancel={vi.fn()} />)

    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Completing…' })).toBeDisabled()
  })
})
