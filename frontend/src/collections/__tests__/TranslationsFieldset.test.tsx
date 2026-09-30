// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, it, expect, vi } from 'vitest'
import { TranslationsFieldset } from '../TranslationsFieldset'
import type { CollectionTexts } from '../types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../../settings/api', () => ({ getTenantSettings: vi.fn(async () => ({ language: 'nl' })), updateTenantSettings: vi.fn() }))

function Stateful() {
  const [value, setValue] = useState<Record<string, CollectionTexts>>({})
  return (
    <>
      <TranslationsFieldset value={value} fallback={{ name: 'Zomer', description: '', seo_title: '', seo_description: '' }} onChange={setValue} />
      <output data-testid="value">{JSON.stringify(value)}</output>
    </>
  )
}

describe('collection TranslationsFieldset', () => {
  it('edits each language other than the company one', async () => {
    const user = userEvent.setup()
    render(<Stateful />)

    expect(await screen.findByRole('tab', { name: 'English' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Nederlands' })).toBeNull()
    expect(screen.getByLabelText('Name (English)')).toHaveAttribute('placeholder', 'Zomer')
    await user.type(screen.getByLabelText('Name (English)'), 'Summer')
    await user.click(screen.getByRole('tab', { name: 'Português (Brasil)' }))
    await user.type(screen.getByLabelText('Search engine title (Português (Brasil))'), 'Verão')

    expect(JSON.parse(screen.getByTestId('value').textContent!)).toEqual({ en: { name: 'Summer' }, 'pt-BR': { seo_title: 'Verão' } })
  })
})
