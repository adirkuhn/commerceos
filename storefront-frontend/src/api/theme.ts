import { request } from './client'
import type { PageSlug, ThemePage } from '../types'

export function getStorefrontTheme(slug: PageSlug): Promise<ThemePage> {
  return request<ThemePage>(`/theme/pages/${slug}`)
}
