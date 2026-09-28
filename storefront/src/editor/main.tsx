import { createRoot } from 'react-dom/client'
import '../theme/editor.css'
import { EditorAPI } from './api'
import { EditorApp } from './EditorApp'
import { takeSessionToken } from './session'

const apiBase = document.querySelector<HTMLMetaElement>('meta[name="cos-api-base"]')?.content ?? ''
const token = takeSessionToken(window.location, window.sessionStorage, url => window.history.replaceState(null, '', url))
const root = createRoot(document.getElementById('editor-root')!)

if (!token || !apiBase) {
  root.render(
    <main className="mx-auto mt-32 max-w-md px-4 text-center">
      <h1 className="text-xl font-semibold">Open the editor from CommerceOS</h1>
      <p className="mt-2 text-neutral-600">Go to Sales channels → your store → Themes, and choose Customize.</p>
    </main>,
  )
} else {
  root.render(<EditorApp api={new EditorAPI({ baseURL: apiBase, token })} />)
}
