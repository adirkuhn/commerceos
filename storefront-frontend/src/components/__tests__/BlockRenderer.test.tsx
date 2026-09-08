// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BlockRenderer } from '../BlockRenderer'
import type { Block } from '../../types'

function block(overrides: Partial<Block>): Block {
  return {
    id: 'b1',
    type: 'hero',
    layout: { desktop: { x: 0, y: 0, width: 100, height: 100 }, mobile: null },
    props: {},
    ...overrides,
  }
}

describe('BlockRenderer', () => {
  it('renders a hero block\'s heading and subheading', () => {
    render(<BlockRenderer block={block({ type: 'hero', props: { heading: 'Big sale', subheading: '50% off' } })} />)

    expect(screen.getByText('Big sale')).toBeInTheDocument()
    expect(screen.getByText('50% off')).toBeInTheDocument()
  })

  it('renders a rich_text block\'s content', () => {
    render(<BlockRenderer block={block({ type: 'rich_text', props: { content: 'Hello shoppers' } })} />)

    expect(screen.getByText('Hello shoppers')).toBeInTheDocument()
  })

  it('renders a custom_html block\'s markup', () => {
    render(<BlockRenderer block={block({ type: 'custom_html', props: { html: '<strong>Sale!</strong>' } })} />)

    expect(screen.getByText('Sale!').tagName).toBe('STRONG')
  })

  it('shows a coming-soon placeholder for a functional block', () => {
    render(<BlockRenderer block={block({ type: 'cart_summary', props: {} })} />)

    expect(screen.getByText(/Cart summary — coming soon/)).toBeInTheDocument()
  })
})
