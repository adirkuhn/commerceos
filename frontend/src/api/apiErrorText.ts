import i18next from 'i18next'
import { CodedApiError } from './codedApiError'

// apiErrorText is what a screen shows for a failed call: an API error
// code's text in the admin's language (errors.json, keyed by the backend's
// domain/errs codes) plus any upstream detail, else the error's own message; undefined for anything
// that isn't an Error, so callers write `apiErrorText(err) ?? t('...')`.
export function apiErrorText(err: unknown): string | undefined {
  if (err instanceof CodedApiError && i18next.exists(`errors:${err.code}`)) {
    const text = i18next.t(`errors:${err.code}` as 'errors:INTERNAL')
    return err.detail ? `${text} (${err.detail})` : text
  }
  return err instanceof Error ? err.message : undefined
}
