import type { ThemeStorage } from './themePreference'
import { appearanceCss, fontStylesheetHref, type AppearanceSettings } from './appearance'

// Also read by the inline script in index.html, which applies the last
// seen appearance before first paint -- keep the key and ids in sync.
export const APPEARANCE_STORAGE_KEY = 'commerceos.appearance'
const STYLE_ID = 'appearance-css'
const FONT_ID = 'appearance-font'

export interface AppliedAppearance {
  css: string
  fontHref: string | null
}

export function renderAppearance(a: AppearanceSettings): AppliedAppearance {
  return { css: appearanceCss(a), fontHref: fontStylesheetHref(a.font) }
}

function upsert<K extends 'style' | 'link'>(doc: Document, tag: K, id: string): HTMLElementTagNameMap[K] {
  const existing = doc.getElementById(id)
  if (existing) {
    // Keep it last in <head>, after the app's own stylesheets.
    if (doc.head.lastElementChild !== existing) doc.head.appendChild(existing)
    return existing as HTMLElementTagNameMap[K]
  }
  const el = doc.createElement(tag)
  el.id = id
  doc.head.appendChild(el)
  return el
}

// applyAppearance puts the shop's look on the page: one unlayered <style>
// overriding the depot tokens, plus the font's stylesheet when the page
// doesn't load it already. The defaults leave no trace.
export function applyAppearance(doc: Document, applied: AppliedAppearance): void {
  if (applied.css) upsert(doc, 'style', STYLE_ID).textContent = applied.css
  else doc.getElementById(STYLE_ID)?.remove()

  if (applied.fontHref) {
    const link = upsert(doc, 'link', FONT_ID)
    link.rel = 'stylesheet'
    if (link.getAttribute('href') !== applied.fontHref) link.href = applied.fontHref
  } else {
    doc.getElementById(FONT_ID)?.remove()
  }
}

// The last applied look, kept per browser so the next visit paints it
// before the API answers. Never authoritative: the API value replaces it.
export function saveAppliedAppearance(storage: ThemeStorage | null, applied: AppliedAppearance): void {
  try {
    storage?.setItem(APPEARANCE_STORAGE_KEY, JSON.stringify(applied))
  } catch {
    // Not cached this time; the next visit waits for the API instead.
  }
}
