import { createRoot } from 'react-dom/client'
import { I18nextProvider } from 'react-i18next'
import '../theme/editor.css'
import { createEditorI18n } from './i18n/createEditorI18n'
import { EditorAPI } from './api'
import { EditorApp } from './EditorApp'
import { editorLocale } from './editorLocale'
import { OpenFromAdmin } from './OpenFromAdmin'
import { takeSessionToken } from './session'
import { takeThemePreference } from './editorLook'

const apiBase = document.querySelector<HTMLMetaElement>('meta[name="cos-api-base"]')?.content ?? ''
// Read before takeSessionToken strips the fragment.
const themePreference = takeThemePreference(window.location.hash, window.sessionStorage)
const token = takeSessionToken(window.location, window.sessionStorage, url => window.history.replaceState(null, '', url))
// The editor's own texts: the browser's language until the edit session
// says which language the admin user works in (useDraft switches then).
// The canvas shows components in the store's language (see CanvasProviders).
const locale = editorLocale()
document.documentElement.lang = locale
const i18n = createEditorI18n(locale)
const root = createRoot(document.getElementById('editor-root')!)

root.render(
  <I18nextProvider i18n={i18n}>
    {token && apiBase ? <EditorApp api={new EditorAPI({ baseURL: apiBase, token })} apiBase={apiBase} themePreference={themePreference} /> : <OpenFromAdmin />}
  </I18nextProvider>,
)
