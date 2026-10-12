import { codedRequest } from '../codedRequest'
import type { Appearance, AppearanceSettings } from '../../theme/appearance'

function url(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/settings/appearance`
}

// GET /settings/appearance: the shop's admin look (defaults, with
// updated_at null, until saved). Rejects with CodedApiError.
export function getAppearance(token: string): Promise<Appearance> {
  return codedRequest<Appearance>('GET', url(), token)
}

// PUT /settings/appearance (owner/admin): full replace. A 422 carries an
// APPEARANCE_* code, e.g. APPEARANCE_ACCENT_LOW_CONTRAST.
export function updateAppearance(token: string, input: AppearanceSettings): Promise<Appearance> {
  return codedRequest<Appearance>('PUT', url(), token, input)
}
