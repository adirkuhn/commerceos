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

  it('keeps polling indefinitely -- a slow-but-working parse never throws a timeout', async () => {
    const fetchJob = vi.fn()
    for (let i = 0; i < 25; i++) {
      fetchJob.mockResolvedValueOnce({ status: 'running', error_message: '' })
    }
    fetchJob.mockResolvedValueOnce({ status: 'completed', error_message: '' })

    const result = await pollParsingJob(fetchJob, { intervalMs: 0 })

    expect(result.status).toBe('completed')
    expect(fetchJob).toHaveBeenCalledTimes(26)
  })

  it('calls onSlow once after slowAfterMs elapses without a terminal status', async () => {
    const fetchJob = vi.fn()
      .mockResolvedValueOnce({ status: 'pending', error_message: '' })
      .mockResolvedValueOnce({ status: 'pending', error_message: '' })
      .mockResolvedValueOnce({ status: 'pending', error_message: '' })
      .mockResolvedValueOnce({ status: 'pending', error_message: '' })
      .mockResolvedValueOnce({ status: 'completed', error_message: '' })
    const onSlow = vi.fn()

    await pollParsingJob(fetchJob, { intervalMs: 10, slowAfterMs: 20, onSlow })

    expect(onSlow).toHaveBeenCalledTimes(1)
  })

  it('does not call onSlow when the job resolves before slowAfterMs', async () => {
    const fetchJob = vi.fn().mockResolvedValueOnce({ status: 'completed', error_message: '' })
    const onSlow = vi.fn()

    await pollParsingJob(fetchJob, { intervalMs: 10, slowAfterMs: 20, onSlow })

    expect(onSlow).not.toHaveBeenCalled()
  })
})
