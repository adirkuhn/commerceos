import type { Editor } from 'grapesjs'
import type { ThemeSettings } from '../../core/document'
import { compileDocumentStyles } from '../../core/style'
import { settingsToCSS } from '../../core/settings'
import { fromGrapes } from './serializer'

const STYLE_ID = 'cos-canvas-styles'

/** Writes the storefront CSS (base, theme settings, node styles) into the
 *  canvas frame -- the same CSS the live storefront gets. */
export function syncCanvasStyles(editor: Editor, baseCSS: string, settings: ThemeSettings) {
  const doc = editor.Canvas.getDocument()
  if (!doc) return
  let el = doc.getElementById(STYLE_ID)
  if (!el) {
    el = doc.createElement('style')
    el.id = STYLE_ID
    doc.head.appendChild(el)
  }
  const nodes = compileDocumentStyles(fromGrapes(editor.getWrapper()!))
  el.textContent = baseCSS + settingsToCSS(settings) + nodes
}
