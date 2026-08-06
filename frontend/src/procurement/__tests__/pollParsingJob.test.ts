import { describe, it, expect, vi } from 'vitest'
import { pollParsingJob } from '../pollParsingJob'
import type { ParsingJobLike } from '../pollParsingJob'

describe('pollParsingJob', () => {
  it('returns immediately when the first fetch is already terminal', async () => {
    const job: ParsingJobLike = { status: 'completed', error_message: '' }
    const fetchJob = vi.fn().mockResolvedValue(job)

    const result = await pollParsingJob(fetchJob, { intervalMs: 0 })

    expect(result).toBe(job)
    expect(fetchJob).toHaveBeenCalledTimes(1)
  })

  it('keeps polling through pending/running until a terminal status arrives', async () => {
    const fetchJob = vi.fn()
      .mockResolvedValueOnce({ status: 'pending', error_message: '' })
      .mockResolvedValueOnce({ status: 'running', error_message: '' })
      .mockResolvedValueOnce({ status: 'failed', error_message: 'AI document parsing is not configured' })

    const result = await pollParsingJob(fetchJob, { intervalMs: 0 })

    expect(fetchJob).toHaveBeenCalledTimes(3)
    expect(result.status).toBe('failed')
    expect(result.error_message).toBe('AI document parsing is not configured')
  })

  it('throws a timeout error when maxAttempts is exhausted without a terminal status', async () => {
    const fetchJob = vi.fn().mockResolvedValue({ status: 'running', error_message: '' })

    await expect(pollParsingJob(fetchJob, { intervalMs: 0, maxAttempts: 3 })).rejects.toThrow(/timed out/i)
    expect(fetchJob).toHaveBeenCalledTimes(3)
  })
})
