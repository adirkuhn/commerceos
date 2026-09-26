// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { ProductEditHeader } from '../ProductEditHeader'

const base = { token: 't', savedProductId: 'p1', name: 'Guarana', showResync: false, saving: false, saveDisabled: false, onClose: vi.fn() }

describe('ProductEditHeader merge action', () => {
  it('offers "Merge into…" when the editor passes onMerge (admins, saved product)', async () => {
    const onMerge = vi.fn()
    render(<ProductEditHeader {...base} onMerge={onMerge} />)
    await userEvent.click(screen.getByRole('button', { name: /Merge into/ }))
    expect(onMerge).toHaveBeenCalled()
  })

  it('hides it otherwise', () => {
    render(<ProductEditHeader {...base} />)
    expect(screen.queryByRole('button', { name: /Merge into/ })).not.toBeInTheDocument()
  })
})
