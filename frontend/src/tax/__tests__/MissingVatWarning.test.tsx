// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import { MissingVatWarning } from '../MissingVatWarning'

describe('MissingVatWarning', () => {
  it('names the products without a VAT rate and links to Settings -> Tax', () => {
    render(<MemoryRouter><MissingVatWarning products={['Flocão de milho', 'Feijão preto']} /></MemoryRouter>)
    expect(screen.getByRole('status')).toHaveTextContent('No VAT rate for Flocão de milho, Feijão preto')
    expect(screen.getByRole('link', { name: 'Set it in Settings → Tax' })).toHaveAttribute('href', '/settings/tax')
  })

  it('shows nothing when every product has a rate', () => {
    const { container } = render(<MemoryRouter><MissingVatWarning products={[]} /></MemoryRouter>)
    expect(container).toBeEmptyDOMElement()
  })
})
