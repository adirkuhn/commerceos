// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SearchField } from '../SearchField'

describe('SearchField', () => {
  it('types and clears, giving focus back to the box', () => {
    const onChange = vi.fn()
    const { rerender } = render(<SearchField label="Search products" value="" onChange={onChange} />)
    expect(screen.queryByRole('button', { name: 'Clear search' })).not.toBeInTheDocument()

    fireEvent.change(screen.getByRole('textbox', { name: 'Search products' }), { target: { value: 'fei' } })
    expect(onChange).toHaveBeenLastCalledWith('fei')

    rerender(<SearchField label="Search products" value="fei" onChange={onChange} />)
    fireEvent.click(screen.getByRole('button', { name: 'Clear search' }))
    expect(onChange).toHaveBeenLastCalledWith('')
    expect(screen.getByRole('textbox', { name: 'Search products' })).toHaveFocus()
  })

  it('Escape clears too', () => {
    const onChange = vi.fn()
    render(<SearchField label="Search products" value="fei" onChange={onChange} />)

    fireEvent.keyDown(screen.getByRole('textbox', { name: 'Search products' }), { key: 'Escape' })

    expect(onChange).toHaveBeenLastCalledWith('')
  })
})
