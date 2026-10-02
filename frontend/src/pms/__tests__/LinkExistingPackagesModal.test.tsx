// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../api', () => ({
  linkExistingWeightPackages: vi.fn(),
}))

import * as pmsApi from '../api'
import { LinkExistingPackagesModal } from '../LinkExistingPackagesModal'
import type { LinkExistingWeightPackagesResult } from '../types'

const preview: LinkExistingWeightPackagesResult = {
  linked: [
    { sku_id: 's380', sku_code: '2994721003809', variant_name: '~ 380g', weight_grams: 380, price_minor_units: 1150, price_changed: false },
    { sku_id: 's528', sku_code: '2994721005285', variant_name: '~ 528g', weight_grams: 528, price_minor_units: 15835, price_changed: true },
  ],
  skipped: [{ sku_id: 'sx', sku_code: '7897122604154', variant_name: 'Gift box', reason: 'not_a_scale_label' }],
  errors: [],
}

beforeEach(() => vi.clearAllMocks())

function renderModal() {
  const onLinked = vi.fn()
  const onClose = vi.fn()
  render(<LinkExistingPackagesModal token="tok" bulkSkuId="sku-pool" currencyCode="EUR" onLinked={onLinked} onClose={onClose} />)
  return { onLinked, onClose }
}

describe('LinkExistingPackagesModal', () => {
  it('previews what would be linked and skipped, changing nothing yet', async () => {
    vi.mocked(pmsApi.linkExistingWeightPackages).mockResolvedValueOnce(preview)
    renderModal()

    expect(await screen.findByText('~ 380g')).toBeInTheDocument()
    expect(pmsApi.linkExistingWeightPackages).toHaveBeenCalledWith('tok', 'sku-pool', true)
    expect(screen.getByText(/was priced per weight/)).toBeInTheDocument()
    expect(screen.getByText('Gift box')).toBeInTheDocument()
    expect(screen.getByText('not a scale label')).toBeInTheDocument()
    expect(pmsApi.linkExistingWeightPackages).toHaveBeenCalledTimes(1)
  })

  it('links on confirm and reports how many', async () => {
    vi.mocked(pmsApi.linkExistingWeightPackages).mockResolvedValueOnce(preview).mockResolvedValueOnce({ ...preview, skipped: [] })
    const { onLinked } = renderModal()

    await userEvent.click(await screen.findByRole('button', { name: 'Link 2 packages' }))

    await waitFor(() => expect(pmsApi.linkExistingWeightPackages).toHaveBeenLastCalledWith('tok', 'sku-pool', false))
    expect(await screen.findByText('2 packages linked.')).toBeInTheDocument()
    expect(onLinked).toHaveBeenCalled()
  })

  it('says when there is nothing to link', async () => {
    vi.mocked(pmsApi.linkExistingWeightPackages).mockResolvedValueOnce({ linked: [], skipped: [], errors: [] })
    renderModal()

    expect(await screen.findByText('No unlinked packages found.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Link \d/ })).not.toBeInTheDocument()
  })

  it('shows a failure', async () => {
    vi.mocked(pmsApi.linkExistingWeightPackages).mockRejectedValueOnce(new Error('source sku must be a weight pool with per-weight pricing and a scale code'))
    renderModal()

    expect(await screen.findByText(/must be a weight pool/)).toBeInTheDocument()
  })

  it('lists per-package errors from the link run', async () => {
    vi.mocked(pmsApi.linkExistingWeightPackages).mockResolvedValueOnce(preview).mockResolvedValueOnce({ linked: [preview.linked[1]], skipped: [], errors: ['2994721003809: boom'] })
    renderModal()

    await userEvent.click(await screen.findByRole('button', { name: 'Link 2 packages' }))

    expect(await screen.findByText('1 package linked.')).toBeInTheDocument()
    expect(screen.getByText('2994721003809: boom')).toBeInTheDocument()
  })
})
