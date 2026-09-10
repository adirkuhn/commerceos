// Placeholder for a real HTTP client (fetch/axios wrapper with base URL,
// auth header, error normalization) once VITE_API_MODE=api points at a
// real backend. Every api/*.ts file below currently forwards straight to
// a repository (src/repositories) instead -- components only ever import
// from api/, never from repositories/ directly, so swapping this in is
// invisible to the rest of the app.
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? ''
export const API_KEY = import.meta.env.VITE_API_KEY ?? ''

export class ApiError extends Error {
  constructor(message: string, public status?: number) {
    super(message)
    this.name = 'ApiError'
  }
}
