import type { Job } from './types'

// A job queued from the Scheduled Jobs page: the scheduler runs it on its
// next poll (within ~60s), so the page waits for last_run_at to move.
export interface QueuedRun {
  lastRunAt: string | null
  queuedAt: number
}

export type QueuedRuns = Record<string, QueuedRun>

// Stop waiting after this long, so a stopped scheduler never leaves a
// button stuck on "Queued".
export const QUEUED_RUN_MAX_WAIT_MS = 3 * 60_000

export const QUEUED_RUN_POLL_MS = 5_000

// stillQueued is the queued runs the scheduler has not picked up yet.
export function stillQueued(queued: QueuedRuns, jobs: Job[], now: number): QueuedRuns {
  const byKey = new Map(jobs.map(j => [j.job_key, j]))
  const out: QueuedRuns = {}
  for (const [key, run] of Object.entries(queued)) {
    const job = byKey.get(key)
    if (!job || job.last_run_at !== run.lastRunAt || now - run.queuedAt >= QUEUED_RUN_MAX_WAIT_MS) continue
    out[key] = run
  }
  return out
}
