export interface ParsingJobLike {
  status: 'pending' | 'running' | 'completed' | 'failed'
  error_message: string
}

const TERMINAL_STATUSES: ParsingJobLike['status'][] = ['completed', 'failed']

export interface PollParsingJobOptions {
  intervalMs?: number
  maxAttempts?: number
}

// Polls fetchJob until it reaches a terminal status (completed/failed) --
// cmd/scheduler picks up parsing jobs on a ~5s cycle, so the default
// interval/attempts give it several cycles to finish before giving up,
// instead of the fixed 6s blind wait this replaces (which never checked
// the job at all, so a failed parse looked identical to a successful one).
export async function pollParsingJob<T extends ParsingJobLike>(
  fetchJob: () => Promise<T>,
  { intervalMs = 1500, maxAttempts = 20 }: PollParsingJobOptions = {},
): Promise<T> {
  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    const job = await fetchJob()
    if (TERMINAL_STATUSES.includes(job.status)) return job
    if (attempt < maxAttempts - 1) {
      await new Promise(resolve => setTimeout(resolve, intervalMs))
    }
  }
  throw new Error('Timed out waiting for the document to finish parsing')
}
