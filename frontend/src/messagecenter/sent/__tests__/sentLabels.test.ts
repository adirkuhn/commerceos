import { describe, it, expect } from 'vitest'
import type { TFunction } from 'i18next'
import { dayEnd, dayStart, sentReason } from '../sentLabels'

const t = ((key: string) => `t:${key}`) as unknown as TFunction<'messages'>

describe('sentReason', () => {
  it('translates skip codes and shows server errors as given', () => {
    expect(sentReason(t, { status: 'skipped', reason: 'no_recipient' })).toBe('t:sent.reason.noRecipient')
    expect(sentReason(t, { status: 'skipped', reason: 'template_disabled' })).toBe('t:sent.reason.templateDisabled')
    expect(sentReason(t, { status: 'failed', reason: '550 mailbox unavailable' })).toBe('550 mailbox unavailable')
    expect(sentReason(t, { status: 'sent', reason: '' })).toBe('')
  })
})

describe('dayStart / dayEnd', () => {
  it('span one whole local day', () => {
    const from = new Date(dayStart('2026-10-12'))
    const to = new Date(dayEnd('2026-10-12'))
    expect(from.getHours()).toBe(0)
    expect(from.getDate()).toBe(12)
    expect(to.getDate()).toBe(13)
    expect(to.getHours()).toBe(0)
  })
})
