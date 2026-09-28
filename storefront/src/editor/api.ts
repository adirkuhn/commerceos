import { ApiError } from '../api/storefront'
import type { DocumentKind, SEO, ThemeDocument, ThemeNode, ThemeSettings, DocumentSummary } from '../core/document'

export interface DraftVersion {
  id: string
  number: number
  state: 'draft' | 'sealed'
  revision: number
  settings: ThemeSettings
}

export interface Draft {
  theme: { id: string; name: string; sales_channel_id: string; is_live: boolean }
  version: DraftVersion
  documents: DocumentSummary[]
  sales_channel_id: string
}

export interface DraftWrite {
  document: ThemeDocument
  revision: number
}

/** ApiError plus the node the backend rejected, for document saves. */
export class EditorApiError extends ApiError {
  readonly nodeId?: string

  constructor(status: number, code: string, message: string, nodeId?: string) {
    super(status, code, message)
    this.nodeId = nodeId
  }
}

export interface EditorAPIConfig {
  baseURL: string
  token: string
  fetch?: typeof fetch
}

/** Client for the storefront-hosted editor API (/theme-editor). */
export class EditorAPI {
  private readonly cfg: EditorAPIConfig

  constructor(cfg: EditorAPIConfig) {
    this.cfg = cfg
  }

  openDraft(): Promise<Draft> {
    return this.call('POST', '/theme-editor/draft')
  }

  getDocument(kind: DocumentKind, key: string): Promise<ThemeDocument> {
    return this.call('GET', `/theme-editor/draft/documents/${kind}/${encodeURIComponent(key)}`)
  }

  saveDocument(kind: DocumentKind, key: string, body: { expected_revision: number; root: ThemeNode; title?: string; seo?: SEO }): Promise<DraftWrite> {
    return this.call('PUT', `/theme-editor/draft/documents/${kind}/${encodeURIComponent(key)}`, body)
  }

  saveSettings(expectedRevision: number, settings: ThemeSettings): Promise<DraftVersion> {
    return this.call('PUT', '/theme-editor/draft/settings', { expected_revision: expectedRevision, settings })
  }

  publish(expectedRevision: number, note?: string): Promise<DraftVersion> {
    return this.call('POST', '/theme-editor/publish', { expected_revision: expectedRevision, ...(note ? { note } : {}) })
  }

  createPreviewSession(): Promise<{ token: string; expires_at: string }> {
    return this.call('POST', '/theme-editor/preview-sessions')
  }

  private async call<T>(method: string, path: string, body?: unknown): Promise<T> {
    const headers = new Headers({ Accept: 'application/json', Authorization: `Bearer ${this.cfg.token}` })
    if (body !== undefined) headers.set('Content-Type', 'application/json')
    const res = await (this.cfg.fetch ?? fetch)(`${this.cfg.baseURL}${path}`, {
      method, headers, body: body === undefined ? undefined : JSON.stringify(body),
    })
    let env: { data: T | null; error: { code?: string; message?: string; node_id?: string } | null } | undefined
    try {
      env = await res.json()
    } catch {
      env = undefined
    }
    if (!res.ok || !env || env.data === null) {
      throw new EditorApiError(res.status, env?.error?.code ?? 'HTTP_ERROR', env?.error?.message ?? `request failed (${res.status})`, env?.error?.node_id)
    }
    return env.data
  }
}
