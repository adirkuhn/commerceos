import { describe, expect, it } from 'vitest'
import { buildStarterHomeBlocks } from '../starterTemplate'

describe('buildStarterHomeBlocks', () => {
  it('returns a hero, an about blurb, the product catalog, and a footer block, in reading order', () => {
    const blocks = buildStarterHomeBlocks('Cantinho Brasileiro')

    expect(blocks.map(b => b.type)).toEqual(['hero', 'rich_text', 'product_grid', 'custom_html'])
    // Reading order top-to-bottom -- each block starts below the one before it.
    for (let i = 1; i < blocks.length; i++) {
      expect(blocks[i].layout.desktop.y).toBeGreaterThan(blocks[i - 1].layout.desktop.y)
    }
  })

  it('uses the given store name in the hero heading and the about blurb', () => {
    const blocks = buildStarterHomeBlocks('Cantinho Brasileiro')

    expect(blocks[0].props.heading).toBe('Cantinho Brasileiro')
    expect(blocks[1].props.content).toContain('Cantinho Brasileiro')
  })

  it('falls back to generic copy when no store name is given', () => {
    const blocks = buildStarterHomeBlocks('')

    expect(blocks[0].props.heading).toBe('Welcome to our store')
  })

  it('gives every block a unique id', () => {
    const blocks = buildStarterHomeBlocks('Store')

    expect(new Set(blocks.map(b => b.id)).size).toBe(blocks.length)
  })
})
