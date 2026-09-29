// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import i18next from 'i18next'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { ConnectionsOverview } from '../ConnectionsOverview'
import type { Connection } from '../types'

afterEach(async () => { await i18next.changeLanguage('en') })

const props = {
  loading: false, error: null, testingId: null, testResult: null,
  onBrowse: vi.fn(), onOpenPanel: vi.fn(), onTest: vi.fn(), onDisconnect: vi.fn(),
}

const woo = { id: 'c1', provider: 'woocommerce', store_url: 'https://mystore.com', status: 'connected' } as Connection

describe('ConnectionsOverview', () => {
  it('shows the empty state when nothing is connected', () => {
    render(<ConnectionsOverview {...props} connections={[]} />)
    expect(screen.getByText('No integrations connected')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Browse integrations' })).toBeInTheDocument()
  })

  it('speaks Portuguese when the admin language is pt-BR', async () => {
    await i18next.changeLanguage('pt-BR')
    render(<ConnectionsOverview {...props} connections={[woo]} />)
    expect(screen.getByText('Integrações conectadas')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Configurar' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Excluir importados' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Desconectar' })).toBeInTheDocument()
  })
})
