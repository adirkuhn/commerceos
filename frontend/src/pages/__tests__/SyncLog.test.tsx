// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import SyncLog from '../SyncLog'
import type { SyncLogEntry } from '../../integrations/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../integrations/api', () => ({
  listAllSyncLogs: vi.fn(),
}))

import * as integrationsApi from '../../integrations/api'

function entry(overrides: Partial<SyncLogEntry> = {}): SyncLogEntry {
  return {
    id: 'log-1', listing_id: 'listing-1', product_id: 'prod-1', connection_id: 'conn-1',
    provider: 'woocommerce', external_id: '23609', direction: 'push', status: 'success',
    error_message: '', fields: { name: 'T-Shirt' }, changes: { name: { from: 'Old', to: 'T-Shirt' } },
    synced_at: '2026-08-04T10:00:00Z', product_name: 'T-Shirt', store_url: 'https://mystore.com',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.listAllSyncLogs).mockResolvedValue({ entries: [], total: 0 })
})

describe('Sync History page', () => {
  it('shows an empty state when there are no syncs', async () => {
    render(<SyncLog />)
    expect(await screen.findByText('No syncs yet')).toBeInTheDocument()
  })

  it('lists sync entries with product, channel, direction and status', async () => {
    vi.mocked(integrationsApi.listAllSyncLogs).mockResolvedValue({ entries: [entry()], total: 1 })
    render(<SyncLog />)

    expect(await screen.findAllByText('T-Shirt')).not.toHaveLength(0)
    expect(screen.getByText(/mystore\.com/)).toBeInTheDocument()
    expect(screen.getByText('success')).toBeInTheDocument()
    expect(screen.getByText('name:')).toBeInTheDocument()
  })

  it('falls back to product_id when product_name is missing', async () => {
    vi.mocked(integrationsApi.listAllSyncLogs).mockResolvedValue({
      entries: [entry({ product_name: undefined })], total: 1,
    })
    render(<SyncLog />)

    expect(await screen.findByText('prod-1')).toBeInTheDocument()
  })

  it('shows the original and new value for each changed field, no click needed', async () => {
    vi.mocked(integrationsApi.listAllSyncLogs).mockResolvedValue({ entries: [entry()], total: 1 })
    render(<SyncLog />)

    expect(await screen.findByText('Old')).toBeInTheDocument()
  })

  it('shows only changed fields, not the full field snapshot', async () => {
    vi.mocked(integrationsApi.listAllSyncLogs).mockResolvedValue({
      entries: [entry({
        fields: { name: 'T-Shirt', description: 'unchanged text' },
        changes: { name: { from: 'Old', to: 'T-Shirt' } },
      })],
      total: 1,
    })
    render(<SyncLog />)

    await screen.findByText('name:')
    expect(screen.queryByText('description:')).not.toBeInTheDocument()
    expect(screen.queryByText('unchanged text')).not.toBeInTheDocument()
  })

  it('shows "No changes" when a sync recorded no field diff', async () => {
    vi.mocked(integrationsApi.listAllSyncLogs).mockResolvedValue({
      entries: [entry({ changes: {} })], total: 1,
    })
    render(<SyncLog />)

    expect(await screen.findByText('No changes')).toBeInTheDocument()
  })

  it('shows the error message for a failed sync', async () => {
    vi.mocked(integrationsApi.listAllSyncLogs).mockResolvedValue({
      entries: [entry({ status: 'error', error_message: 'woo unreachable', changes: {} })], total: 1,
    })
    render(<SyncLog />)

    expect(await screen.findByText('woo unreachable')).toBeInTheDocument()
  })

  it('surfaces a fetch error', async () => {
    vi.mocked(integrationsApi.listAllSyncLogs).mockRejectedValue(new Error('internal server error'))
    render(<SyncLog />)

    expect(await screen.findByText('internal server error')).toBeInTheDocument()
  })

  it('paginates when there are more than 20 entries', async () => {
    vi.mocked(integrationsApi.listAllSyncLogs).mockImplementation(async (_token, params = {}) => {
      const page = params.page ?? 1
      return {
        entries: [entry({ id: `log-${page}`, product_name: `Product page ${page}` })],
        total: 25,
      }
    })
    render(<SyncLog />)

    await screen.findByText('Product page 1')
    await userEvent.click(screen.getByRole('button', { name: 'Next page' }))
    await waitFor(() => expect(screen.getByText('Product page 2')).toBeInTheDocument())
  })
})
