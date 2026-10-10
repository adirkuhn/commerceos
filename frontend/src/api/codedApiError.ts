// An API failure that keeps the envelope's error code (e.g.
// PRODUCT_HAS_HISTORY), not just its message, so a screen can react to a
// specific failure -- place it on a field, offer an alternative action, or
// show it translated (see apiErrorText). detail is an upstream provider's
// own reason, when the backend sends one.
export class CodedApiError extends Error {
  constructor(readonly code: string, message: string, readonly detail?: string) {
    super(message)
    this.name = 'CodedApiError'
  }

  // fromEnvelope builds the error from a response envelope's error object.
  static fromEnvelope(error: { code?: string; message?: string; detail?: string } | null | undefined, status: number): CodedApiError {
    return new CodedApiError(error?.code ?? 'HTTP_ERROR', error?.message ?? `HTTP ${status}`, error?.detail || undefined)
  }
}
