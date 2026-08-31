// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { openShipmentLabel, loadShipmentLabelIntoTab } from '../openLabel'

vi.mock('../api', () => ({
  getShipmentLabel: vi.fn(),
}))

import * as shippingApi from '../api'

function fakeTab() {
  return { location: { href: '' }, close: vi.fn() } as unknown as Window
}

beforeEach(() => {
  vi.clearAllMocks()
  URL.createObjectURL = vi.fn(() => 'blob:mock-url')
})

describe('loadShipmentLabelIntoTab', () => {
  it('navigates the given tab to the fetched PDF blob URL', async () => {
    const blob = new Blob(['%PDF-fake'], { type: 'application/pdf' })
    vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(blob)
    const tab = fakeTab()

    const result = await loadShipmentLabelIntoTab(tab, 'tok', 's-1', 'A4')

    expect(shippingApi.getShipmentLabel).toHaveBeenCalledWith('tok', 's-1', 'A4')
    expect(tab.location.href).toBe('blob:mock-url')
    expect(result).toBe(tab)
  })

  it('closes the tab and downloads instead for a non-PDF (ZPL) label', async () => {
    const blob = new Blob(['^XA^XZ'], { type: 'application/vnd.shipment_label+zpl' })
    vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(blob)
    const tab = fakeTab()
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const result = await loadShipmentLabelIntoTab(tab, 'tok', 's-1')

    expect(tab.close).toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalled()
    expect(result).toBeNull()
  })

  it('falls back to a download when no tab was available', async () => {
    const blob = new Blob(['%PDF-fake'], { type: 'application/pdf' })
    vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(blob)
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const result = await loadShipmentLabelIntoTab(null, 'tok', 's-1')

    expect(clickSpy).toHaveBeenCalled()
    expect(result).toBeNull()
  })

  it('closes the tab and rethrows when the fetch fails', async () => {
    vi.mocked(shippingApi.getShipmentLabel).mockRejectedValue(new Error('shipment not found'))
    const tab = fakeTab()

    await expect(loadShipmentLabelIntoTab(tab, 'tok', 's-1')).rejects.toThrow('shipment not found')
    expect(tab.close).toHaveBeenCalled()
  })
})

describe('openShipmentLabel', () => {
  it('opens a blank tab synchronously before fetching the label', async () => {
    const blob = new Blob(['%PDF-fake'], { type: 'application/pdf' })
    vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(blob)
    const tab = fakeTab()
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(tab)

    await openShipmentLabel('tok', 's-1', 'A6')

    expect(openSpy).toHaveBeenCalledWith('', '_blank')
    expect(tab.location.href).toBe('blob:mock-url')
  })
})
