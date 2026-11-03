import { describe, it, expect } from 'vitest'
import { stillQueued, QUEUED_RUN_MAX_WAIT_MS, type QueuedRuns } from '../queuedRuns'
import type { Job } from '../types'

function job(key: string, lastRunAt: string | null): Job {
  return {
    job_key: key, enabled: true, interval_minutes: 60, last_run_at: lastRunAt,
    last_run_status: 'success', last_run_error: '', next_run_at: '2026-11-03T10:00:00Z',
  }
}

const queuedAt = Date.parse('2026-11-03T09:00:00Z')

describe('stillQueued', () => {
  it('keeps a job whose last run has not changed yet', () => {
    const queued: QueuedRuns = { a: { lastRunAt: '2026-11-02T00:00:00Z', queuedAt } }

    expect(stillQueued(queued, [job('a', '2026-11-02T00:00:00Z')], queuedAt + 30_000)).toEqual(queued)
  })

  it('drops a job once the scheduler has run it', () => {
    const queued: QueuedRuns = { a: { lastRunAt: '2026-11-02T00:00:00Z', queuedAt } }

    expect(stillQueued(queued, [job('a', '2026-11-03T09:00:40Z')], queuedAt + 45_000)).toEqual({})
  })

  it('drops a never-run job once it has a first run', () => {
    const queued: QueuedRuns = { a: { lastRunAt: null, queuedAt } }

    expect(stillQueued(queued, [job('a', '2026-11-03T09:00:40Z')], queuedAt + 45_000)).toEqual({})
  })

  it('gives up waiting after the cap so the button never stays stuck', () => {
    const queued: QueuedRuns = { a: { lastRunAt: null, queuedAt } }

    expect(stillQueued(queued, [job('a', null)], queuedAt + QUEUED_RUN_MAX_WAIT_MS)).toEqual({})
  })

  it('drops a job that is no longer listed', () => {
    const queued: QueuedRuns = { a: { lastRunAt: null, queuedAt } }

    expect(stillQueued(queued, [], queuedAt + 1000)).toEqual({})
  })
})
