// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { DiscrepancyRow } from '../DiscrepancyRow'
import { DateFormatProvider } from '../../format/DateFormatProvider'
import type { ReceivingDiscrepancy } from '../types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../../settings/api', () => ({ getTenantSettings: vi.fn() }))
import * as settingsApi from '../../settings/api'
import type { TenantSettings } from '../../settings/types'

const expiryDiscrepancy: ReceivingDiscrepancy = {
  id: 'd-1', tenant_id: 't-1', receiving_record_id: 'r-1', line_item_id: 'li-1',
  type: 'expiry_mismatch', expected_qty: 10, received_qty: 10,
  details: 'expiry date is within 30 days (or already past)', expiry_date: '2026-10-03',
  status: 'open', created_at: '2026-09-26T10:00:00Z',
}

function renderRow(d: ReceivingDiscrepancy, token = 'tok') {
  return render(
    <DateFormatProvider token={token}>
      <DiscrepancyRow
        discrepancy={d} allDiscrepancies={[d]} lineItem={undefined} warehouseId={null} warehouses={[]}
        onAssignWarehouse={async () => {}} invoiceFileName="inv.csv" disabled={false} onReload={() => {}}
      />
    </DateFormatProvider>,
  )
}

beforeEach(() => vi.clearAllMocks())

describe('DiscrepancyRow expiry date', () => {
  it("renders the expiry date in the tenant's date format", async () => {
    vi.mocked(settingsApi.getTenantSettings).mockResolvedValue(
      { date_format: 'MM/DD/YYYY', timezone: 'Europe/Amsterdam' } as TenantSettings,
    )
    renderRow(expiryDiscrepancy)

    expect(await screen.findByText(/Expires 10\/03\/2026/)).toBeTruthy()
    expect(screen.getByText(/expiry date is within 30 days \(or already past\)/)).toBeTruthy()
  })

  it('uses the default DD/MM/YYYY until settings load', () => {
    vi.mocked(settingsApi.getTenantSettings).mockReturnValue(new Promise(() => {}))
    renderRow(expiryDiscrepancy)

    expect(screen.getByText(/Expires 03\/10\/2026/)).toBeTruthy()
  })

  it('shows no expiry line when the discrepancy has no expiry date', () => {
    vi.mocked(settingsApi.getTenantSettings).mockReturnValue(new Promise(() => {}))
    renderRow({ ...expiryDiscrepancy, type: 'quantity_mismatch', expiry_date: null, details: 'expected 10, invoice shows 8' })

    expect(screen.queryByText(/Expires/)).toBeNull()
  })
})
