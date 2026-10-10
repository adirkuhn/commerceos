import type { i18n as I18n } from 'i18next'
import { EditorApiError } from './api'

// editorErrorText is what the theme editor shows for a failed call: the
// API error code's text in the editor's language (editor.json errors.*,
// mirroring the admin's errors.json for theme errors), else the error's
// own message -- e.g. a page-validation detail naming the broken block --
// else fallback (already translated by the caller).
export function editorErrorText(err: unknown, i18n: I18n, fallback: string): string {
  if (err instanceof EditorApiError) {
    // Codes come from the API (kept by i18next-parser.config's keepRemoved).
    const key = `editor:errors.${err.code}`
    if (i18n.exists(key)) return i18n.t(key as 'editor:errors.INTERNAL')
  }
  return err instanceof Error ? err.message : fallback
}
