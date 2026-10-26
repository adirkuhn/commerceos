// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { ContactPersonsCard } from '../ContactPersonsCard'
import type { ContactPerson } from '../../types'

const kylian: ContactPerson = { id: 'cp-1', customer_id: 'c-1', first_name: 'Kylian', last_name: 'Klockenbrink', role: 'Employee', email: 'k@claus.nl', phone: '', created_at: '' }

function renderCard(over: Partial<React.ComponentProps<typeof ContactPersonsCard>> = {}) {
  const props = { persons: [kylian], error: null, onAdd: vi.fn().mockResolvedValue(undefined), onUpdate: vi.fn().mockResolvedValue(undefined), onRemove: vi.fn().mockResolvedValue(undefined), ...over }
  render(<ContactPersonsCard {...props} />)
  return props
}

describe('ContactPersonsCard', () => {
  it('lists each contact person with role and email', () => {
    renderCard()
    const item = screen.getByRole('listitem')
    expect(item).toHaveTextContent('Kylian Klockenbrink')
    expect(item).toHaveTextContent('Employee')
    expect(item).toHaveTextContent('k@claus.nl')
  })

  it('adds a contact person; only a name is needed', async () => {
    const { onAdd } = renderCard({ persons: [] })
    expect(screen.getByText('No contact persons yet.')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Add contact person' }))
    const save = screen.getByRole('button', { name: 'Save' })
    expect(save).toBeDisabled()
    fireEvent.change(screen.getByLabelText('First name'), { target: { value: 'Sara' } })
    fireEvent.change(screen.getByLabelText('Role'), { target: { value: 'Purchasing' } })
    fireEvent.click(save)

    await waitFor(() => expect(onAdd).toHaveBeenCalledWith({ first_name: 'Sara', last_name: '', role: 'Purchasing', email: '', phone: '' }))
    await waitFor(() => expect(screen.queryByLabelText('First name')).not.toBeInTheDocument())
  })

  it('edits a contact person', async () => {
    const { onUpdate } = renderCard()
    fireEvent.click(screen.getByRole('button', { name: 'Edit Kylian Klockenbrink' }))
    expect(screen.getByLabelText('Email')).toHaveValue('k@claus.nl')
    fireEvent.change(screen.getByLabelText('Phone'), { target: { value: '0612' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(onUpdate).toHaveBeenCalledWith('cp-1', expect.objectContaining({ first_name: 'Kylian', phone: '0612' })))
  })

  it('removes a contact person after confirming', async () => {
    const { onRemove } = renderCard()
    fireEvent.click(screen.getByRole('button', { name: 'Edit Kylian Klockenbrink' }))
    fireEvent.click(screen.getByRole('button', { name: 'Remove' }))
    fireEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: 'Remove' }))

    await waitFor(() => expect(onRemove).toHaveBeenCalledWith('cp-1'))
  })

  it('shows why a save failed', async () => {
    renderCard({ persons: [], onAdd: vi.fn().mockRejectedValue(new Error('boom')) })
    fireEvent.click(screen.getByRole('button', { name: 'Add contact person' }))
    fireEvent.change(screen.getByLabelText('First name'), { target: { value: 'Sara' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })
})
