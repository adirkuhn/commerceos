// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

vi.mock('../quotesApi', () => ({ setAutomaticCostPrice: vi.fn() }))
import * as api from '../quotesApi'
import { CostCell } from '../CostCell'

const row = { sku_id: 's-1', sku_code: 'FEI-1', currency_code: 'EUR', cost_price_minor_units: 500 }

beforeEach(() => vi.clearAllMocks())

function renderCell(r = row) {
  const onSaved = vi.fn()
  render(<table><tbody><tr><CostCell token="tok" row={r} onSaved={onSaved} /></tr></tbody></table>)
  return { onSaved }
}

describe('CostCell', () => {
  it('saves a typed cost for the SKU and tells the table', async () => {
    vi.mocked(api.setAutomaticCostPrice).mockResolvedValue()
    const { onSaved } = renderCell()
    expect(screen.getByText('€5.00')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Set cost of FEI-1' }))
    const input = screen.getByRole('textbox', { name: 'Cost of FEI-1' })
    expect(input).toHaveValue('5.00')
    fireEvent.change(input, { target: { value: '4,25' } })
    fireEvent.keyDown(input, { key: 'Enter' })

    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(api.setAutomaticCostPrice).toHaveBeenCalledWith('tok', 's-1', 425)
    expect(screen.queryByRole('textbox', { name: 'Cost of FEI-1' })).not.toBeInTheDocument()
  })

  it('sets a cost where none is known', () => {
    renderCell({ ...row, cost_price_minor_units: null } as unknown as typeof row)
    expect(screen.getByText('—')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Set cost of FEI-1' }))
    expect(screen.getByRole('textbox', { name: 'Cost of FEI-1' })).toHaveValue('')
  })

  it('Escape cancels without saving', () => {
    renderCell()
    fireEvent.click(screen.getByRole('button', { name: 'Set cost of FEI-1' }))
    fireEvent.keyDown(screen.getByRole('textbox', { name: 'Cost of FEI-1' }), { key: 'Escape' })

    expect(api.setAutomaticCostPrice).not.toHaveBeenCalled()
    expect(screen.getByText('€5.00')).toBeInTheDocument()
  })

  it('does not send an invalid cost', () => {
    renderCell()
    fireEvent.click(screen.getByRole('button', { name: 'Set cost of FEI-1' }))
    fireEvent.change(screen.getByRole('textbox', { name: 'Cost of FEI-1' }), { target: { value: 'abc' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save cost' }))

    expect(api.setAutomaticCostPrice).not.toHaveBeenCalled()
  })

  it('shows a failed save and stays in edit mode', async () => {
    vi.mocked(api.setAutomaticCostPrice).mockRejectedValue(new Error('boom'))
    const { onSaved } = renderCell()
    fireEvent.click(screen.getByRole('button', { name: 'Set cost of FEI-1' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save cost' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
    expect(onSaved).not.toHaveBeenCalled()
    expect(screen.getByRole('textbox', { name: 'Cost of FEI-1' })).toBeInTheDocument()
  })
})
