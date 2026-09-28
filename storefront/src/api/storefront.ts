import type { DocumentKind, StorefrontTheme, ThemeDocument } from '../core/document'

export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

export interface StorefrontAPIConfig {
  /** e.g. https://api.example.com/api/v1 -- the SSR server uses an internal URL. */
  baseURL: string
  /** The channel's publishable storefront key. */
  storeKey: string
  /** Preview-session token: the draft is shown instead of the live theme. */
  previewToken?: string
  fetch?: typeof fetch
}

interface Envelope<T> {
  data: T | null
  error: { code?: string; message?: string } | null
}

/** Client for the public storefront API (/storefront/v1). */
export class StorefrontAPI {
  private readonly cfg: StorefrontAPIConfig

  constructor(cfg: StorefrontAPIConfig) {
    this.cfg = cfg
  }

  get previewing(): boolean {
    return Boolean(this.cfg.previewToken)
  }

  getTheme(): Promise<StorefrontTheme> {
    return this.get('/storefront/v1/theme')
  }

  getDocument(theme: StorefrontTheme, kind: DocumentKind, key: string): Promise<ThemeDocument> {
    const tail = `${kind}/${encodeURIComponent(key)}`
    return theme.preview
      ? this.get(`/storefront/v1/theme/preview/documents/${tail}`)
      : this.get(`/storefront/v1/theme/versions/${encodeURIComponent(theme.version_id)}/documents/${tail}`)
  }

  private async get<T>(path: string): Promise<T> {
    const headers = new Headers({ Accept: 'application/json', Authorization: `Bearer ${this.cfg.storeKey}` })
    if (this.cfg.previewToken) headers.set('X-Theme-Preview', this.cfg.previewToken)
    const doFetch = this.cfg.fetch ?? fetch
    const res = await doFetch(`${this.cfg.baseURL}${path}`, { headers })
    let body: Envelope<T> | undefined
    try {
      body = (await res.json()) as Envelope<T>
    } catch {
      body = undefined
    }
    if (!res.ok || !body || body.data === null) {
      throw new ApiError(res.status, body?.error?.code ?? 'HTTP_ERROR', body?.error?.message ?? `request failed (${res.status})`)
    }
    return body.data
  }
}
