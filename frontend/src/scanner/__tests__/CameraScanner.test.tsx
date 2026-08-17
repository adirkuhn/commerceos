// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { CameraScanner } from '../CameraScanner'

const mockStop = vi.fn()
const mockDecodeFromVideoDevice = vi.fn()

vi.mock('@zxing/browser', () => ({
  BrowserMultiFormatReader: vi.fn().mockImplementation(() => ({
    decodeFromVideoDevice: mockDecodeFromVideoDevice,
  })),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockDecodeFromVideoDevice.mockResolvedValue({ stop: mockStop })
})

describe('CameraScanner', () => {
  it('starts the video decoder on mount', async () => {
    render(<CameraScanner onDecode={() => {}} onClose={() => {}} />)

    await waitFor(() => expect(mockDecodeFromVideoDevice).toHaveBeenCalled())
    const [deviceId, videoEl] = mockDecodeFromVideoDevice.mock.calls[0]
    expect(deviceId).toBeUndefined()
    expect(videoEl).toBeInstanceOf(HTMLVideoElement)
  })

  it('calls onDecode and stops the stream once a code is found', async () => {
    const onDecode = vi.fn()
    render(<CameraScanner onDecode={onDecode} onClose={() => {}} />)
    await waitFor(() => expect(mockDecodeFromVideoDevice).toHaveBeenCalled())

    const callback = mockDecodeFromVideoDevice.mock.calls[0][2]
    callback({ getText: () => 'DECODED-123' }, undefined, { stop: mockStop })

    expect(onDecode).toHaveBeenCalledWith('DECODED-123')
    expect(mockStop).toHaveBeenCalled()
  })

  it('ignores callback ticks with no result (still scanning)', async () => {
    const onDecode = vi.fn()
    render(<CameraScanner onDecode={onDecode} onClose={() => {}} />)
    await waitFor(() => expect(mockDecodeFromVideoDevice).toHaveBeenCalled())

    const callback = mockDecodeFromVideoDevice.mock.calls[0][2]
    callback(undefined, new Error('not found'), { stop: mockStop })

    expect(onDecode).not.toHaveBeenCalled()
  })

  it('shows an error if the camera fails to start', async () => {
    mockDecodeFromVideoDevice.mockRejectedValue(new Error('Permission denied'))

    render(<CameraScanner onDecode={() => {}} onClose={() => {}} />)

    expect(await screen.findByText('Permission denied')).toBeInTheDocument()
  })

  it('Close stops the stream and calls onClose', async () => {
    const onClose = vi.fn()
    render(<CameraScanner onDecode={() => {}} onClose={onClose} />)
    await waitFor(() => expect(mockDecodeFromVideoDevice).toHaveBeenCalled())

    await userEvent.click(screen.getByTitle('Close'))

    expect(onClose).toHaveBeenCalled()
  })

  it('stops the stream on unmount', async () => {
    const { unmount } = render(<CameraScanner onDecode={() => {}} onClose={() => {}} />)
    await waitFor(() => expect(mockDecodeFromVideoDevice).toHaveBeenCalled())

    unmount()

    expect(mockStop).toHaveBeenCalled()
  })

  it('defaults to a square viewfinder frame with a scan-line animation', () => {
    render(<CameraScanner onDecode={() => {}} onClose={() => {}} />)

    expect(screen.getByTestId('scan-frame')).toHaveAttribute('data-shape', 'square')
    expect(screen.getByTestId('scan-line')).toBeInTheDocument()
  })

  it('renders a rect viewfinder frame when frameShape="rect" (barcode scans)', () => {
    render(<CameraScanner onDecode={() => {}} onClose={() => {}} frameShape="rect" />)

    expect(screen.getByTestId('scan-frame')).toHaveAttribute('data-shape', 'rect')
    expect(screen.getByTestId('scan-line')).toBeInTheDocument()
  })
})
