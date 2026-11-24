import type { ThemeStorage } from './themePreference'
import type { Look } from './lookPreference'
import { appearanceCss, type AppearanceSettings } from './appearance'

// Also read by the inline script in index.html, which applies the last
// seen appearance before first paint -- keep the key, id and shape in sync.
export const APPEARANCE_STORAGE_KEY = 'commerceos.appearance'
const STYLE_ID = 'appearance-css'
// Left behind by the old font setting.
const OLD_FONT_ID = 'appearance-font'

export interface AppliedAppearance {
  css: string
  // The shop's default look.
  look: Look
}

export function renderAppearance(a: AppearanceSettings): AppliedAppearance {
  return { css: appearanceCss(a), look: a.default_look }
}

// applyAppearance puts the shop's overrides on the page: one unlayered
// <style>, kept last in <head> after the app's own stylesheets. The
// defaults leave no trace. (The look itself is applied by useLook.)
export function applyAppearance(doc: Document, applied: AppliedAppearance): void {
  doc.getElementById(OLD_FONT_ID)?.remove()
  if (!applied.css) {
    doc.getElementById(STYLE_ID)?.remove()
    return
  }
  let style = doc.getElementById(STYLE_ID)
  if (!style) {
    style = doc.createElement('style')
    style.id = STYLE_ID
  }
  if (doc.head.lastElementChild !== style) doc.head.appendChild(style)
  style.textContent = applied.css
}

// The last applied appearance, kept per browser so the next visit paints it
// before the API answers. Never authoritative: the API value replaces it.
export function saveAppliedAppearance(storage: ThemeStorage | null, applied: AppliedAppearance): void {
  try {
    storage?.setItem(APPEARANCE_STORAGE_KEY, JSON.stringify({ css: applied.css, look: applied.look }))
  } catch {
    // Not cached this time; the next visit waits for the API instead.
  }
}
