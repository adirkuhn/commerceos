// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { GroupForm } from '../components/GroupForm'

afterEach(async () => { await i18next.changeLanguage('en') })

describe('GroupForm', () => {
  it('submits the new group', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    render(<GroupForm onSubmit={onSubmit} onCancel={vi.fn()} />)

    expect(screen.getByText('New group')).toBeInTheDocument()
    await userEvent.type(screen.getByRole('textbox', { name: 'Name *' }), 'Packers')
    await userEvent.click(screen.getByRole('button', { name: 'Create' }))

    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ name: 'Packers' }))
  })

  it('speaks Portuguese when the admin language is pt-BR', async () => {
    await i18next.changeLanguage('pt-BR')
    render(<GroupForm onSubmit={vi.fn()} onCancel={vi.fn()} />)

    expect(screen.getByText('Novo grupo')).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: 'Nome *' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Criar' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Cancelar' })).toBeInTheDocument()
  })
})
