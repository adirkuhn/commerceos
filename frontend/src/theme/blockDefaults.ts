import type { Block, BlockType } from './types'

function newBlockId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `block-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

const DEFAULT_PROPS: Record<BlockType, Record<string, unknown>> = {
  hero: { heading: 'Welcome', subheading: '', imageUrl: '' },
  product_grid: {},
  rich_text: { content: 'Edit this text…' },
  custom_html: { html: '' },
  product_details: {},
  cart_summary: {},
  checkout_form: {},
  order_status: {},
}

export function createBlock(type: BlockType): Block {
  return {
    id: newBlockId(),
    type,
    layout: { desktop: { x: 20, y: 20, width: 320, height: 160 }, mobile: null },
    props: { ...DEFAULT_PROPS[type] },
  }
}
