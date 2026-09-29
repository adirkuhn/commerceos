import { resolveLocale, type Locale } from '../i18n/locale'

/** The editor's language: the edit session's (the admin user's), else the
 *  browser's until the session is known. */
export function editorLocale(sessionLocale?: string | null): Locale {
  return resolveLocale(sessionLocale || (typeof navigator === 'undefined' ? undefined : navigator.language))
}
