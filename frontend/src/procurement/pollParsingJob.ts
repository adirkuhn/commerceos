export interface ParsingJobLike {
  status: 'pending' | 'running' | 'completed' | 'failed'
  error_message: string
}

const TERMINAL_STATUSES: ParsingJobLike['status'][] = ['completed', 'failed']

export interface PollParsingJobOptions {
  intervalMs?: number
  // How long to wait with no terminal status before firing onSlow once, so
  // the UI can say "still working" instead of a bare spinner.
  slowAfterMs?: number
  onSlow?: () => void
}

// Polls fetchJob until it reaches a terminal status (completed/failed).
// cmd/scheduler picks up parsing jobs on a ~5s cycle and AI extraction has
// no fixed upper bound, so this polls indefinitely rather than giving up
// after N attempts -- a slow-but-working parse should never surface as an
// error just because the frontend got impatient.
export async function pollParsingJob<T extends ParsingJobLike>(
  fetchJob: () => Promise<T>,
  { intervalMs = 1500, slowAfterMs = 20000, onSlow }: PollParsingJobOptions = {},
): Promise<T> {
  let elapsedMs = 0
  let slowNoticeFired = false
  for (;;) {
    const job = await fetchJob()
    if (TERMINAL_STATUSES.includes(job.status)) return job
    if (!slowNoticeFired && elapsedMs >= slowAfterMs) {
      slowNoticeFired = true
      onSlow?.()
    }
    await new Promise(resolve => setTimeout(resolve, intervalMs))
    elapsedMs += intervalMs
  }
}
