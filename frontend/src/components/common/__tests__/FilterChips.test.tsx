// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { FilterChips } from '../FilterChips'

const OPTIONS = [{ value: 'woo', label: 'WooCommerce' }, { value: 'bol', label: 'Bol' }]

describe('FilterChips', () => {
  it('is a labelled group of toggle buttons showing which are on', () => {
    render(<FilterChips label="Channel" options={OPTIONS} selected={['bol']} onToggle={() => {}} />)
    const group = screen.getByRole('group', { name: 'Channel' })
    expect(group).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Bol' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'WooCommerce' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('shows the label above the chips', () => {
    render(<FilterChips label="Channel" options={OPTIONS} selected={[]} onToggle={() => {}} />)
    expect(screen.getByText('Channel')).toBeVisible()
  })

  it('toggles the clicked option', async () => {
    const onToggle = vi.fn()
    render(<FilterChips label="Channel" options={OPTIONS} selected={[]} onToggle={onToggle} />)
    await userEvent.click(screen.getByRole('button', { name: 'WooCommerce' }))
    expect(onToggle).toHaveBeenCalledWith('woo')
  })
})
