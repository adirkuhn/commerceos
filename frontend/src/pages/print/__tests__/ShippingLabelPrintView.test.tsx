// @vitest-environment jsdom
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ShippingLabelPrintView } from '../ShippingLabelPrintView'

vi.mock('../../../shipping/api', () => ({
  getShipmentLabel: vi.fn(),
}))

import * as shippingApi from '../../../shipping/api'

beforeEach(() => {
  vi.clearAllMocks()
  URL.createObjectURL = vi.fn(() => 'blob:mock-url')
  URL.revokeObjectURL = vi.fn()
  window.print = vi.fn()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('ShippingLabelPrintView', () => {
  it('fetches the label and renders a PDF embed pointed at the blob URL', async () => {
    const blob = new Blob(['%PDF-fake'], { type: 'application/pdf' })
    vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(blob)

    const { container } = render(<ShippingLabelPrintView token="tok" shipmentId="s-1" onClose={vi.fn()} />)

    await waitFor(() => expect(container.querySelector('embed')).toBeInTheDocument())
    const embed = container.querySelector('embed')!
    expect(embed).toHaveAttribute('src', 'blob:mock-url')
    expect(embed).toHaveAttribute('type', 'application/pdf')
    expect(shippingApi.getShipmentLabel).toHaveBeenCalledWith('tok', 's-1', 'A4')
  })

  it('defaults to A4, and switching to A6 re-fetches at the new size without a new shipment', async () => {
    const blobA4 = new Blob(['%PDF-a4'], { type: 'application/pdf' })
    const blobA6 = new Blob(['%PDF-a6'], { type: 'application/pdf' })
    vi.mocked(shippingApi.getShipmentLabel).mockResolvedValueOnce(blobA4).mockResolvedValueOnce(blobA6)

    const { container } = render(<ShippingLabelPrintView token="tok" shipmentId="s-1" onClose={vi.fn()} />)
    await waitFor(() => expect(container.querySelector('embed')).toBeInTheDocument())
    expect(screen.getByRole('button', { name: 'A4' })).toHaveAttribute('aria-pressed', 'true')

    fireEvent.click(screen.getByRole('button', { name: 'A6' }))

    await waitFor(() => expect(shippingApi.getShipmentLabel).toHaveBeenCalledWith('tok', 's-1', 'A6'))
    expect(screen.getByRole('button', { name: 'A6' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('honors an explicit defaultSize prop instead of always starting at A4', async () => {
    const blob = new Blob(['%PDF-fake'], { type: 'application/pdf' })
    vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(blob)

    render(<ShippingLabelPrintView token="tok" shipmentId="s-1" defaultSize="A6" onClose={vi.fn()} />)

    await waitFor(() => expect(shippingApi.getShipmentLabel).toHaveBeenCalledWith('tok', 's-1', 'A6'))
  })

  it('calls window.print once the embedded PDF has loaded (not before)', async () => {
    const blob = new Blob(['%PDF-fake'], { type: 'application/pdf' })
    vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(blob)

    const { container } = render(<ShippingLabelPrintView token="tok" shipmentId="s-1" onClose={vi.fn()} />)
    await waitFor(() => expect(container.querySelector('embed')).toBeInTheDocument())

    expect(window.print).not.toHaveBeenCalled()
    fireEvent.load(container.querySelector('embed')!)
    expect(window.print).toHaveBeenCalledTimes(1)
  })

  it('shows a download link instead of an embed for a non-PDF (ZPL) label', async () => {
    const blob = new Blob(['^XA^XZ'], { type: 'application/vnd.shipment_label+zpl' })
    vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(blob)

    render(<ShippingLabelPrintView token="tok" shipmentId="s-1" onClose={vi.fn()} />)

    expect(await screen.findByText(/no visual preview/i)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /download the zpl file/i })).toBeInTheDocument()
    expect(window.print).not.toHaveBeenCalled()
  })

  it('shows an error message when fetching the label fails', async () => {
    vi.mocked(shippingApi.getShipmentLabel).mockRejectedValue(new Error('shipment not found'))

    render(<ShippingLabelPrintView token="tok" shipmentId="missing" onClose={vi.fn()} />)

    expect(await screen.findByText('shipment not found')).toBeInTheDocument()
  })

  it('calls onClose when the close button is clicked', async () => {
    const blob = new Blob(['%PDF-fake'], { type: 'application/pdf' })
    vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(blob)
    const onClose = vi.fn()

    const { container } = render(<ShippingLabelPrintView token="tok" shipmentId="s-1" onClose={onClose} />)
    await waitFor(() => expect(container.querySelector('embed')).toBeInTheDocument())
    fireEvent.click(screen.getByTitle('Close'))

    expect(onClose).toHaveBeenCalled()
  })
})
