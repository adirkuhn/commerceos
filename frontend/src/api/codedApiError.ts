// An API failure that keeps the envelope's error code (e.g.
// PRODUCT_HAS_HISTORY), not just its message, so a screen can react to a
// specific failure -- place it on a field, or offer an alternative action.
export class CodedApiError extends Error {
  constructor(readonly code: string, message: string) {
    super(message)
    this.name = 'CodedApiError'
  }
}
