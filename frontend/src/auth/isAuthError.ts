// isAuthError reports whether an API error means the session is no longer
// valid (401 / unauthorized) -- screens use it to send the user back to
// pairing/login instead of showing a generic error.
export function isAuthError(err: unknown): boolean {
  return err instanceof Error && /401|unauthorized/i.test(err.message)
}
