import { describe, it, expect } from 'vitest'
import { fullName } from '../personName'

describe('fullName', () => {
  it('joins first and last name', () => {
    expect(fullName({ first_name: 'Ana', last_name: 'Silva', email: 'a@x.com' })).toBe('Ana Silva')
  })
  it('uses whichever name part exists', () => {
    expect(fullName({ first_name: '', last_name: 'Silva', email: 'a@x.com' })).toBe('Silva')
  })
  it('falls back to the email when there is no name', () => {
    expect(fullName({ first_name: '', last_name: '', email: 'a@x.com' })).toBe('a@x.com')
  })
})
