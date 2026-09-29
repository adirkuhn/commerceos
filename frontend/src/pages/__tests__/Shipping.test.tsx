// @vitest-environment jsdom
import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { act } from 'react'
import { describe, it, expect } from 'vitest'
import i18next from 'i18next'
import Shipping from '../Shipping'
import { renderAt } from '../../test/renderAt'

// Shipping is still backed by hardcoded demo data; these tests cover only
// that its view state lives in the URL.
describe('Shipping URLs (refresh + Back)', () => {
  it('the Rules tab is /shipping/rules, and Back returns to shipments', async () => {
    const { router } = renderAt(<Shipping />, '/shipping')
    await userEvent.click(await screen.findByRole('button', { name: 'Carrier rules' }))
    expect(router.state.location.pathname).toBe('/shipping/rules')

    await act(() => router.navigate(-1))
    expect(router.state.location.pathname).toBe('/shipping')
  })

  it('renders its texts in pt-BR', async () => {
    await i18next.changeLanguage('pt-BR')
    try {
      renderAt(<Shipping />, '/shipping')
      expect(await screen.findByRole('button', { name: 'Regras de transportadora' })).toBeInTheDocument()
      expect(screen.getByPlaceholderText('Pedido, cliente, rastreio...')).toBeInTheDocument()
      expect(screen.getByText('Enviados hoje')).toBeInTheDocument()
    } finally {
      await i18next.changeLanguage('en')
    }
  })

  it('/shipping/rules opens the rules tab on load', async () => {
    renderAt(<Shipping />, '/shipping', '/shipping/rules')
    expect(await screen.findByText('DE · < 5kg')).toBeInTheDocument()
    // the shipments tab's search box is not shown on the rules tab
    expect(screen.queryByPlaceholderText('Order, customer, tracking...')).not.toBeInTheDocument()
  })

  it('search lives in ?q=', async () => {
    const { router } = renderAt(<Shipping />, '/shipping', '/shipping')
    const input = await screen.findByPlaceholderText('Order, customer, tracking...')
    await userEvent.type(input, 'ab')
    expect(router.state.location.search).toBe('?q=ab')
  })
})
