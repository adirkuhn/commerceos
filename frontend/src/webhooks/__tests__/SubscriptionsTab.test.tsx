// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import i18next from 'i18next'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { SubscriptionsTab } from '../SubscriptionsTab'

afterEach(async () => { await i18next.changeLanguage('en') })

const props = { webhooks: [], loading: false, error: null, remove: vi.fn(), refresh: vi.fn(), onRegisterWebhook: vi.fn() }

describe('SubscriptionsTab', () => {
  it('says when nothing is registered', () => {
    render(<SubscriptionsTab {...props} />)
    expect(screen.getByText('Nothing registered yet at any connection.')).toBeInTheDocument()
  })

  it('speaks Portuguese when the admin language is pt-BR', async () => {
    await i18next.changeLanguage('pt-BR')
    render(<SubscriptionsTab {...props} />)
    expect(screen.getByText('Assinaturas ativas nos provedores')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Registrar um webhook' })).toBeInTheDocument()
    expect(screen.getByText('Nada cadastrado ainda em nenhuma conexão.')).toBeInTheDocument()
  })
})
