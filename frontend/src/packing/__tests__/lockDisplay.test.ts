import { describe, it, expect } from 'vitest'
import { canTakeOverLocks, formatLockAge, shortUserId } from '../lockDisplay'

describe('canTakeOverLocks', () => {
  it('mirrors the backend admin roles', () => {
    expect(canTakeOverLocks('super_admin')).toBe(true)
    expect(canTakeOverLocks('customer_owner')).toBe(true)
    expect(canTakeOverLocks('customer_admin')).toBe(true)
    expect(canTakeOverLocks('customer_user')).toBe(false)
    expect(canTakeOverLocks(undefined)).toBe(false)
  })
})

describe('formatLockAge', () => {
  const now = new Date('2026-09-24T10:00:00Z').getTime()
  it('rounds to minutes and hours', () => {
    expect(formatLockAge('2026-09-24T09:59:40Z', now)).toBe('<1 min')
    expect(formatLockAge('2026-09-24T09:48:00Z', now)).toBe('12 min')
    expect(formatLockAge('2026-09-24T08:55:00Z', now)).toBe('1 h 5 min')
    expect(formatLockAge('2026-09-24T08:00:00Z', now)).toBe('2 h')
  })
  it('never goes negative on clock skew', () => {
    expect(formatLockAge('2026-09-24T10:05:00Z', now)).toBe('<1 min')
  })
})

describe('shortUserId', () => {
  it('labels an unknown user by a short id', () => {
    expect(shortUserId('3f2a9c1e-aaaa-bbbb')).toBe('User 3f2a9c')
  })
})
