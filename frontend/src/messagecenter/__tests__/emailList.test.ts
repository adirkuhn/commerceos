import { describe, it, expect } from 'vitest'
import { parseEmailList } from '../emailList'

describe('parseEmailList', () => {
  it('splits on new lines, commas and semicolons, trimming blanks', () => {
    expect(parseEmailList(' a@example.nl \n\nb@example.nl, c@example.nl;d@example.nl ')).toEqual([
      'a@example.nl', 'b@example.nl', 'c@example.nl', 'd@example.nl',
    ])
  })

  it('is empty for blank text', () => {
    expect(parseEmailList('  \n ')).toEqual([])
  })
})
