// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ScheduledJobs from '../ScheduledJobs'
import type { Job, SyncScheduleRow } from '../../scheduler/types'

let mockRole = 'super_admin'
vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: { role: mockRole }, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../scheduler/api', () => ({
  listJobs: vi.fn(),
  runJobNow: vi.fn(),
  listSyncSchedules: vi.fn(),
  runSyncScheduleNow: vi.fn(),
}))

import * as schedulerApi from '../../scheduler/api'

function job(overrides: Partial<Job> = {}): Job {
  return {
    job_key: 'cost_price_recalculation', enabled: true, interval_minutes: 1440,
    last_run_at: '2026-08-20T00:00:00Z', last_run_status: 'success', last_run_error: '',
    next_run_at: '2026-08-21T00:00:00Z',
    ...overrides,
  }
}

function scheduleRow(overrides: Partial<SyncScheduleRow> = {}): SyncScheduleRow {
  return {
    id: 'sched-1', tenant_id: 'tenant-1', connection_id: 'conn-1', connection_store_url: 'https://mystore.com',
    entity_type: 'orders', direction: 'import', enabled: true, interval_minutes: 1,
    last_run_at: '2026-08-20T00:00:00Z', last_run_status: 'success', last_run_error: '', next_run_at: '2026-08-20T00:01:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  mockRole = 'super_admin'
  vi.mocked(schedulerApi.listJobs).mockResolvedValue([])
  vi.mocked(schedulerApi.listSyncSchedules).mockResolvedValue([])
})

describe('ScheduledJobs', () => {
  it('shows an access-denied message for a non-super_admin user, without calling the API', async () => {
    mockRole = 'customer_owner'

    render(<ScheduledJobs />)

    expect(await screen.findByText('Super admin access required.')).toBeInTheDocument()
    expect(schedulerApi.listJobs).not.toHaveBeenCalled()
  })

  it('lists global jobs and sync schedules for a super_admin', async () => {
    vi.mocked(schedulerApi.listJobs).mockResolvedValue([job()])
    vi.mocked(schedulerApi.listSyncSchedules).mockResolvedValue([scheduleRow()])

    render(<ScheduledJobs />)

    expect(await screen.findByText('cost_price_recalculation')).toBeInTheDocument()
    expect(screen.getByText('https://mystore.com')).toBeInTheDocument()
  })

  it('runs a global job now and refreshes the list', async () => {
    vi.mocked(schedulerApi.listJobs).mockResolvedValue([job({ last_run_status: '' })])
    vi.mocked(schedulerApi.runJobNow).mockResolvedValue(null)

    render(<ScheduledJobs />)
    await screen.findByText('cost_price_recalculation')

    await userEvent.click(screen.getAllByRole('button', { name: 'Run now' })[0])

    expect(schedulerApi.runJobNow).toHaveBeenCalledWith('test-token', 'cost_price_recalculation')
    expect(schedulerApi.listJobs).toHaveBeenCalledTimes(2)
  })

  it('runs a sync schedule now', async () => {
    vi.mocked(schedulerApi.listSyncSchedules).mockResolvedValue([scheduleRow()])
    vi.mocked(schedulerApi.runSyncScheduleNow).mockResolvedValue(null)

    render(<ScheduledJobs />)
    await screen.findByText('https://mystore.com')

    await userEvent.click(screen.getAllByRole('button', { name: 'Run now' })[0])

    expect(schedulerApi.runSyncScheduleNow).toHaveBeenCalledWith('test-token', 'sched-1')
  })

  it('disables Run now for a disabled sync schedule', async () => {
    vi.mocked(schedulerApi.listSyncSchedules).mockResolvedValue([scheduleRow({ enabled: false })])

    render(<ScheduledJobs />)
    await screen.findByText('https://mystore.com')

    expect(screen.getByRole('button', { name: 'Run now' })).toBeDisabled()
  })

  it('surfaces a load error', async () => {
    vi.mocked(schedulerApi.listJobs).mockRejectedValue(new Error('forbidden'))

    render(<ScheduledJobs />)

    expect(await screen.findByText('forbidden')).toBeInTheDocument()
  })
})
