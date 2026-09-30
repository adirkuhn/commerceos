// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ProductTranslationsCard } from '../ProductTranslationsCard'
import type { Product } from '../types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({ setProductTranslations: vi.fn() }))
vi.mock('../../settings/api', () => ({ getTenantSettings: vi.fn(async () => ({ language: 'pt-BR' })), updateTenantSettings: vi.fn() }))
import * as api from '../api'

const product = {
  id: 'p1', name: 'Pão de queijo', description: 'Quentinho', slug: 'pao-de-queijo',
  translations: { en: { name: 'Cheese bread' } },
} as unknown as Product

beforeEach(() => vi.clearAllMocks())

describe('ProductTranslationsCard', () => {
  it('offers the languages other than the company one, prefilled with saved translations', async () => {
    render(<ProductTranslationsCard product={product} />)

    expect(await screen.findByRole('tab', { name: 'English' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', { name: 'Nederlands' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Português (Brasil)' })).toBeNull()
    expect(screen.getByLabelText('Name')).toHaveValue('Cheese bread')
    expect(screen.getByLabelText('Description')).toHaveAttribute('placeholder', 'Quentinho')
  })

  it('saves every language\'s texts', async () => {
    vi.mocked(api.setProductTranslations).mockResolvedValue({ ...product, translations: { en: { name: 'Cheese bread' }, nl: { name: 'Kaasbroodje' } } })
    const user = userEvent.setup()
    render(<ProductTranslationsCard product={product} />)

    await user.click(await screen.findByRole('tab', { name: 'Nederlands' }))
    await user.type(screen.getByLabelText('Name'), 'Kaasbroodje')
    await user.click(screen.getByRole('button', { name: 'Save translations' }))

    expect(api.setProductTranslations).toHaveBeenCalledWith('tok', 'p1', { en: { name: 'Cheese bread' }, nl: { name: 'Kaasbroodje' } })
    expect(await screen.findByText('Translations saved.')).toBeInTheDocument()
  })

  it('shows a save error', async () => {
    vi.mocked(api.setProductTranslations).mockRejectedValue(new Error('invalid product translation'))
    const user = userEvent.setup()
    render(<ProductTranslationsCard product={product} />)
    await user.click(await screen.findByRole('button', { name: 'Save translations' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('invalid product translation')
  })
})
