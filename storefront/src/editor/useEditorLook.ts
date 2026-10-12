import { useEffect } from 'react'
import { editorAppearanceCss, editorFontHref, isDarkLook, type EditorAppearance, type ThemePreference } from './editorLook'

function upsert<K extends 'style' | 'link'>(tag: K, id: string): HTMLElementTagNameMap[K] {
  const existing = document.getElementById(id)
  if (existing) return existing as HTMLElementTagNameMap[K]
  const el = document.createElement(tag)
  el.id = id
  document.head.appendChild(el)
  return el
}

/** Puts the editor in the admin's light/dark appearance (following the OS
 *  live for "system") and applies the shop's font, accent and corners once
 *  the draft brings them. */
export function useEditorLook(preference: ThemePreference, appearance: EditorAppearance | undefined): void {
  useEffect(() => {
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const apply = () => document.documentElement.classList.toggle('dark', isDarkLook(preference, media.matches))
    apply()
    if (preference !== 'system') return
    media.addEventListener('change', apply)
    return () => media.removeEventListener('change', apply)
  }, [preference])

  useEffect(() => {
    if (!appearance) return
    const font = upsert('link', 'editor-font')
    font.rel = 'stylesheet'
    font.href = editorFontHref(appearance.font)
    const css = editorAppearanceCss(appearance)
    if (css) upsert('style', 'editor-appearance').textContent = css
    else document.getElementById('editor-appearance')?.remove()
  }, [appearance])
}
