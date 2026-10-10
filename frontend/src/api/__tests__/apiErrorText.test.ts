import { describe, it, expect, afterEach } from 'vitest'
import i18next from 'i18next'
import { apiErrorText } from '../apiErrorText'
import { CodedApiError } from '../codedApiError'

afterEach(async () => { await i18next.changeLanguage('en') })

describe('apiErrorText', () => {
  it('translates a known error code into the admin language', async () => {
    i18next.addResource('nl', 'errors', 'TEST_THING_GONE', 'Ding is weg')
    await i18next.changeLanguage('nl')
    expect(apiErrorText(new CodedApiError('TEST_THING_GONE', 'thing is gone'))).toBe('Ding is weg')
  })

  it('shows the message for a code without a translation yet', () => {
    expect(apiErrorText(new CodedApiError('SOMETHING_NEW', 'something new happened'))).toBe('something new happened')
  })

  it('shows a plain error’s message, and nothing for a non-error', () => {
    expect(apiErrorText(new Error('network down'))).toBe('network down')
    expect(apiErrorText('boom')).toBeUndefined()
  })

  it('translates generic server failures', () => {
    expect(apiErrorText(new CodedApiError('INTERNAL', 'internal server error'))).toBe('Something went wrong on our side. Please try again.')
  })
})
