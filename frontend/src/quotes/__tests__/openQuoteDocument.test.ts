// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../../settings/company/api', () => ({ getCompanyProfile: vi.fn() }))
import { getCompanyProfile } from '../../settings/company/api'
import { openQuoteDocument } from '../openQuoteDocument'

const doc = { reference: 'Q-0007', name: 'Sabor', customerName: null, date: '24-10-2026', currency: 'EUR', lines: [] }

function fakeTab() {
  const written: string[] = []
  return {
    written,
    tab: { document: { open: vi.fn(), write: (h: string) => written.push(h), close: vi.fn() }, close: vi.fn() },
  }
}

beforeEach(() => vi.clearAllMocks())

describe('openQuoteDocument', () => {
  it('opens a new tab at once and writes the quote with the seller into it', async () => {
    const { tab, written } = fakeTab()
    const open = vi.spyOn(window, 'open').mockReturnValue(tab as never)
    vi.mocked(getCompanyProfile).mockResolvedValue({ legal_name: 'Cantinho B.V.', address: {} } as never)

    const pending = openQuoteDocument('tok', doc)
    expect(open).toHaveBeenCalledWith('', '_blank')
    await pending

    expect(written[written.length - 1]).toContain('Cantinho B.V.')
    expect(written[written.length - 1]).toContain('Q-0007')
    expect(tab.document.close).toHaveBeenCalled()
  })

  it('tells the user when the browser blocked the tab', async () => {
    vi.spyOn(window, 'open').mockReturnValue(null)
    await expect(openQuoteDocument('tok', doc)).rejects.toThrow(/pop-ups/)
  })

  it('closes the tab again when the company profile cannot be loaded', async () => {
    const { tab } = fakeTab()
    vi.spyOn(window, 'open').mockReturnValue(tab as never)
    vi.mocked(getCompanyProfile).mockRejectedValue(new Error('boom'))

    await expect(openQuoteDocument('tok', doc)).rejects.toThrow('boom')
    expect(tab.close).toHaveBeenCalled()
  })
})
