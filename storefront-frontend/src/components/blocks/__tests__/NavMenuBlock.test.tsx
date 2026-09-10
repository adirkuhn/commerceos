// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { NavMenuBlock } from '../NavMenuBlock'

function renderBlock(links: { label: string; url: string }[]) {
  return render(<MemoryRouter><NavMenuBlock links={links} /></MemoryRouter>)
}

describe('NavMenuBlock', () => {
  it('renders links in the given order', () => {
    renderBlock([{ label: 'Bebidas', url: '/products?category=cat-1' }, { label: 'Snacks', url: '/products?category=cat-2' }])

    const links = screen.getAllByRole('link')
    expect(links.map(l => l.textContent)).toEqual(['Bebidas', 'Snacks'])
  })

  it('renders an internal path as a client-side Link', () => {
    renderBlock([{ label: 'Products', url: '/products' }])

    expect(screen.getByRole('link', { name: 'Products' })).toHaveAttribute('href', '/products')
  })

  it('renders an external url as a plain, new-tab anchor', () => {
    renderBlock([{ label: 'WhatsApp', url: 'https://wa.me/+31640140482' }])

    const link = screen.getByRole('link', { name: 'WhatsApp' })
    expect(link).toHaveAttribute('href', 'https://wa.me/+31640140482')
    expect(link).toHaveAttribute('target', '_blank')
  })

  it('renders nothing when given no links', () => {
    renderBlock([])

    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })
})
