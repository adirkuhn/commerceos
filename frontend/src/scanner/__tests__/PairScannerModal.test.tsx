// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PairScannerModal } from '../PairScannerModal'
import type { ScannerPairingCode } from '../types'

vi.mock('../api', () => ({
  generatePairingCode: vi.fn(),
  getPairingCodeStatus: vi.fn(),
}))

import * as api from '../api'

function pairing(overrides: Partial<ScannerPairingCode> = {}): ScannerPairingCode {
  return {
    id: 'pairing-1', tenant_id: 'tenant-1', code: '123456', token: 'opaque-token-abc',
    status: 'pending', expires_at: '2026-08-05T00:05:00Z', created_at: '2026-08-05T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('PairScannerModal', () => {
  it('defaults to showing the QR code', async () => {
    vi.mocked(api.generatePairingCode).mockResolvedValue(pairing())

    render(<PairScannerModal token="desktop-token" onClose={() => {}} />)

    expect(await screen.findByRole('img', { name: 'Scan to pair' })).toBeInTheDocument()
  })

  it('lists tabs in order: QR Code, Barcode, Digits', async () => {
    vi.mocked(api.generatePairingCode).mockResolvedValue(pairing())

    render(<PairScannerModal token="desktop-token" onClose={() => {}} />)
    await screen.findByRole('img', { name: 'Scan to pair' })

    const tabs = screen.getAllByRole('button').map(b => b.textContent)
    expect(tabs.filter(t => ['QR Code', 'Barcode', 'Digits'].includes(t ?? ''))).toEqual(['QR Code', 'Barcode', 'Digits'])
  })

  it('shows a barcode encoding the token when the Barcode tab is selected', async () => {
    vi.mocked(api.generatePairingCode).mockResolvedValue(pairing())

    render(<PairScannerModal token="desktop-token" onClose={() => {}} />)
    await screen.findByRole('button', { name: 'Barcode' })

    await userEvent.click(screen.getByRole('button', { name: 'Barcode' }))

    await waitFor(() => expect(document.querySelector('svg.jsbarcode')).toBeInTheDocument())
  })

  it('shows the digit code when the Digits tab is selected', async () => {
    vi.mocked(api.generatePairingCode).mockResolvedValue(pairing())

    render(<PairScannerModal token="desktop-token" onClose={() => {}} />)
    await screen.findByRole('button', { name: 'Digits' })

    await userEvent.click(screen.getByRole('button', { name: 'Digits' }))

    expect(screen.getByText('123456')).toBeInTheDocument()
  })

  it('still polls for claim status regardless of which tab is active', async () => {
    vi.mocked(api.generatePairingCode).mockResolvedValue(pairing())
    vi.mocked(api.getPairingCodeStatus).mockResolvedValue(pairing({ status: 'claimed' }))

    render(<PairScannerModal token="desktop-token" onClose={() => {}} />)
    await screen.findByRole('img', { name: 'Scan to pair' })

    expect(await screen.findByText('Scanner paired', {}, { timeout: 3000 })).toBeInTheDocument()
  })
})
