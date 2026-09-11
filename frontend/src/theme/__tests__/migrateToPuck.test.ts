import { describe, expect, it, vi, beforeEach } from 'vitest'
import { migrateAllChannelsToPuck, migrateChannelToPuck } from '../migrateToPuck'
import type { ThemePage } from '../types'
import type { SalesChannel } from '../../integrations/types'

vi.mock('../api', () => ({ getThemePage: vi.fn(), saveThemePage: vi.fn() }))
import * as themeApi from '../api'

function channel(overrides: Partial<SalesChannel> = {}): SalesChannel {
  return {
    id: 'chan-1', name: 'B2C Store', type: 'internal', connection_id: null,
    enrollment_mode: 'all', active: true, created_at: '', updated_at: '', ...overrides,
  }
}

function page(overrides: Partial<ThemePage> = {}): ThemePage {
  return { sales_channel_id: 'chan-1', slug: 'home', canvas_width: 1440, blocks: [], custom_css: '', editor_mode: 'freeform', ...overrides }
}

const heroBlock: ThemePage['blocks'][number] = {
  id: 'b1', type: 'hero', layout: { desktop: { x: 0, y: 0, width: 100, height: 100 }, mobile: null }, props: { heading: 'Welcome' },
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.stubEnv('VITE_STOREFRONT_URL', 'https://shop.test')
})

describe('migrateChannelToPuck', () => {
  it('converts every freeform page with real content to puck_data + editor_mode puck', async () => {
    vi.mocked(themeApi.getThemePage).mockImplementation(async (_t, _c, slug) =>
      slug === 'home' ? page({ slug: 'home', blocks: [heroBlock] }) : page({ slug }))
    vi.mocked(themeApi.saveThemePage).mockResolvedValue(page())

    const result = await migrateChannelToPuck('token', channel())

    expect(result.migratedSlugs).toEqual(['home'])
    expect(themeApi.saveThemePage).toHaveBeenCalledTimes(1)
    expect(themeApi.saveThemePage).toHaveBeenCalledWith('token', 'chan-1', 'home', expect.objectContaining({
      editor_mode: 'puck',
      puck_data: expect.objectContaining({ content: expect.arrayContaining([expect.objectContaining({ type: 'Hero' })]) }),
    }))
  })

  it('skips a page with no blocks -- nothing to convert', async () => {
    vi.mocked(themeApi.getThemePage).mockResolvedValue(page({ blocks: [] }))

    const result = await migrateChannelToPuck('token', channel())

    expect(result.migratedSlugs).toEqual([])
    expect(themeApi.saveThemePage).not.toHaveBeenCalled()
  })

  it('skips a page already on Puck', async () => {
    vi.mocked(themeApi.getThemePage).mockResolvedValue(page({ blocks: [heroBlock], editor_mode: 'puck' }))

    const result = await migrateChannelToPuck('token', channel())

    expect(result.migratedSlugs).toEqual([])
    expect(themeApi.saveThemePage).not.toHaveBeenCalled()
  })

  it('keeps going after one page fails, and still returns the channel\'s preview URL', async () => {
    vi.mocked(themeApi.getThemePage).mockImplementation(async (_t, _c, slug) => {
      if (slug === 'header') throw new Error('network error')
      return slug === 'home' ? page({ slug: 'home', blocks: [heroBlock] }) : page({ slug })
    })
    vi.mocked(themeApi.saveThemePage).mockResolvedValue(page())

    const result = await migrateChannelToPuck('token', channel({ id: 'chan-2', name: 'Store 2' }))

    expect(result.migratedSlugs).toEqual(['home'])
    expect(result.previewUrl).toBe('https://shop.test/store/enter/chan-2')
  })
})

describe('migrateAllChannelsToPuck', () => {
  it('only migrates internal channels, skipping WooCommerce/Bol/WhatsApp', async () => {
    vi.mocked(themeApi.getThemePage).mockResolvedValue(page({ blocks: [heroBlock] }))
    vi.mocked(themeApi.saveThemePage).mockResolvedValue(page())

    const channels = [
      channel({ id: 'chan-int', type: 'internal' }),
      channel({ id: 'chan-woo', type: 'woocommerce' }),
    ]
    const results = await migrateAllChannelsToPuck('token', channels)

    expect(results.map(r => r.channelId)).toEqual(['chan-int'])
  })
})
