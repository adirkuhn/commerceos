// Hand-written mirror of backend/api/theme/openapi.yaml (admin part).

export type VersionState = 'draft' | 'sealed'
export type SessionPurpose = 'edit' | 'preview'

/** Starting points for a new theme (backend theme presets). */
export const THEME_PRESETS = ['default', 'mercado'] as const
export type ThemePreset = (typeof THEME_PRESETS)[number]

export interface Theme {
  id: string
  sales_channel_id: string
  name: string
  preset_key: string
  published_version_id?: string | null
  is_live: boolean
  created_at: string
  updated_at: string
}

export interface VersionSummary {
  id: string
  number: number
  state: VersionState
  revision: number
  note: string
  created_by: string
  created_at: string
  sealed_at?: string | null
}

export interface SessionToken {
  token: string
  expires_at: string
}
