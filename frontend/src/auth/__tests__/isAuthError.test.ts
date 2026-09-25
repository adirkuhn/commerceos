import { describe, it, expect } from 'vitest'
import { isAuthError } from '../isAuthError'

describe('isAuthError', () => {
  it('detects a 401 / unauthorized error', () => {
    expect(isAuthError(new Error('HTTP 401'))).toBe(true)
    expect(isAuthError(new Error('Unauthorized'))).toBe(true)
  })
  it('ignores other errors and non-errors', () => {
    expect(isAuthError(new Error('HTTP 500'))).toBe(false)
    expect(isAuthError('401')).toBe(false)
  })
})
