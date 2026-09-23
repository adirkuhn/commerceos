import { describe, it, expect } from 'vitest'
import { countryOptions } from '../countries'

describe('countryOptions', () => {
  it('names countries and sorts by name', () => {
    const options = countryOptions()
    expect(options.find(o => o.code === 'NL')?.name).toBe('Netherlands')
    const names = options.map(o => o.name)
    expect(names).toEqual([...names].sort((a, b) => a.localeCompare(b)))
  })

  it('keeps an unlisted saved country as an option', () => {
    expect(countryOptions('JP').some(o => o.code === 'JP')).toBe(true)
    expect(countryOptions().some(o => o.code === 'JP')).toBe(false)
  })
})
