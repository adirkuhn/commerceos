// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

vi.mock('../../search/api', () => ({ globalSearch: vi.fn() }))
import * as searchApi from '../../search/api'
import { CustomerPicker } from '../CustomerPicker'

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(searchApi.globalSearch).mockResolvedValue({
    products: [], suppliers: [], sales_orders: [],
    customers: [{ id: 'c-1', first_name: 'Ana', last_name: 'Souza', email: 'ana@sabor.nl' }, { id: 'c-2', first_name: '', last_name: '', email: 'shop@x.nl' }],
  })
})

describe('CustomerPicker', () => {
  it('searches customers and picks one', async () => {
    const onChange = vi.fn()
    render(<CustomerPicker token="tok" value={null} onChange={onChange} />)

    fireEvent.change(screen.getByRole('textbox', { name: 'Customer' }), { target: { value: 'ana' } })
    fireEvent.click(await screen.findByRole('button', { name: /Ana Souza/ }))

    expect(searchApi.globalSearch).toHaveBeenCalledWith('tok', 'ana')
    expect(onChange).toHaveBeenCalledWith({ id: 'c-1', name: 'Ana Souza', email: 'ana@sabor.nl' })
  })

  it('names a customer without a name by email', async () => {
    render(<CustomerPicker token="tok" value={null} onChange={vi.fn()} />)
    fireEvent.change(screen.getByRole('textbox', { name: 'Customer' }), { target: { value: 'shop' } })
    expect(await screen.findByRole('button', { name: /shop@x\.nl/ })).toBeInTheDocument()
  })

  it('shows the chosen customer and clears it', () => {
    const onChange = vi.fn()
    render(<CustomerPicker token="tok" value={{ id: 'c-1', name: 'Ana Souza', email: 'ana@sabor.nl' }} onChange={onChange} />)

    expect(screen.getByText('Ana Souza')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Remove customer' }))
    expect(onChange).toHaveBeenCalledWith(null)
  })

  it('does not search for a single letter', async () => {
    render(<CustomerPicker token="tok" value={null} onChange={vi.fn()} />)
    fireEvent.change(screen.getByRole('textbox', { name: 'Customer' }), { target: { value: 'a' } })
    await new Promise(r => setTimeout(r, 400))
    await waitFor(() => expect(searchApi.globalSearch).not.toHaveBeenCalled())
  })
})
