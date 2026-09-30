import { ApiError } from '../api/storefront'
import type { AssetRef, DocumentKind, DocumentTexts, SEO, ThemeDocument, ThemeNode, ThemeSettings, DocumentSummary } from '../core/document'

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
  assets?: Record<string, AssetRef>
  /** The editing admin user's language (set when the edit session was
   *  created); the editor's own texts use it. */
  locale?: string
}

export interface Asset {
  id: string
  url: string
  filename: string
  content_type: string
  bytes: number
  width: number
  height: number
  alt: string
  created_at: string
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

  saveDocument(kind: DocumentKind, key: string, body: { expected_revision: number; root: ThemeNode; title?: string; seo?: SEO; translations?: Record<string, DocumentTexts> }): Promise<DraftWrite> {
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

  async listAssets(page = 1, perPage = 60): Promise<{ items: Asset[]; total: number }> {
    const items = await this.call<Asset[]>('GET', `/theme-editor/assets?page=${page}&per_page=${perPage}`)
    return { items, total: this.lastTotal }
  }

  uploadAsset(file: File, alt = ''): Promise<Asset> {
    const form = new FormData()
    form.append('file', file)
    if (alt) form.append('alt', alt)
    return this.call('POST', '/theme-editor/assets', form)
  }

  updateAsset(id: string, alt: string): Promise<Asset> {
    return this.call('PATCH', `/theme-editor/assets/${id}`, { alt })
  }

  async deleteAsset(id: string): Promise<void> {
    await this.call<unknown>('DELETE', `/theme-editor/assets/${id}`, undefined, true)
  }

  private lastTotal = 0

  private async call<T>(method: string, path: string, body?: unknown, allowNullData = false): Promise<T> {
    const headers = new Headers({ Accept: 'application/json', Authorization: `Bearer ${this.cfg.token}` })
    const isForm = typeof FormData !== 'undefined' && body instanceof FormData
    if (body !== undefined && !isForm) headers.set('Content-Type', 'application/json')
    const res = await (this.cfg.fetch ?? fetch)(`${this.cfg.baseURL}${path}`, {
      method, headers, body: body === undefined ? undefined : isForm ? body : JSON.stringify(body),
    })
    let env: { data: T | null; total?: number; error: { code?: string; message?: string; node_id?: string } | null } | undefined
    try {
      env = await res.json()
    } catch {
      env = undefined
    }
    if (res.ok && env && env.data === null && allowNullData) return env.data as T
    if (!res.ok || !env || env.data === null) {
      throw new EditorApiError(res.status, env?.error?.code ?? 'HTTP_ERROR', env?.error?.message ?? `request failed (${res.status})`, env?.error?.node_id)
    }
    this.lastTotal = env.total ?? 0
    return env.data
  }
}
