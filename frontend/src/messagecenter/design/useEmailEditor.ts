import { useEffect, useRef, useState } from 'react'
import grapesjs, { type Editor, type ProjectData } from 'grapesjs'
import presetNewsletter from 'grapesjs-preset-newsletter'
import nl from 'grapesjs/locale/nl.mjs'
import pt from 'grapesjs/locale/pt.mjs'
import 'grapesjs/dist/css/grapes.min.css'
import type { EditorBlock } from './editorBlocks'

export interface EmailEditorOptions {
  // design reopens a saved project; without one, html seeds the canvas.
  design?: Record<string, unknown> | null
  html: string
  blocks: EditorBlock[]
  blockCategory: string
  locale: string
  uploadImage: (file: File) => Promise<string>
  onUploadError: (err: unknown) => void
}

export interface EmailEditor {
  ready: boolean
  // content is the email body with inline styles (what is sent) and the
  // project data to reopen it.
  content: () => { html: string; design: Record<string, unknown> }
  insertText: (text: string) => void
  insertHTML: (html: string) => void
}

const INLINE_HTML = 'gjs-get-inlined-html'

// useEmailEditor mounts GrapesJS with the newsletter preset (email-safe
// blocks, inline CSS on export) plus the CommerceOS blocks into container.
export function useEmailEditor(container: React.RefObject<HTMLDivElement | null>, opts: EmailEditorOptions): EmailEditor {
  const editorRef = useRef<Editor | null>(null)
  const optsRef = useRef(opts)
  optsRef.current = opts
  const [ready, setReady] = useState(false)

  useEffect(() => {
    if (!container.current) return
    const o = optsRef.current
    const editor = grapesjs.init({
      container: container.current,
      height: '100%',
      storageManager: false,
      fromElement: false,
      i18n: { locale: o.locale.slice(0, 2), messages: { nl, pt } },
      plugins: [presetNewsletter],
      pluginsOpts: { [presetNewsletter as unknown as string]: { inlineCss: true, showBlocksOnLoad: true } },
      assetManager: {
        upload: false,
        uploadFile: async (e: DragEvent | Event) => {
          const files = ((e as DragEvent).dataTransfer?.files ?? (e.target as HTMLInputElement | null)?.files) ?? null
          for (const file of Array.from(files ?? [])) {
            try {
              editor.AssetManager.add(await optsRef.current.uploadImage(file))
            } catch (err) {
              optsRef.current.onUploadError(err)
            }
          }
        },
      },
      projectData: (o.design as ProjectData | null | undefined) ?? { pages: [{ component: o.html }] },
    })
    for (const b of o.blocks) {
      editor.BlockManager.add(b.id, { label: b.label, media: b.media, category: o.blockCategory, content: b.content })
    }
    editorRef.current = editor
    editor.on('load', () => setReady(true))
    return () => {
      editorRef.current = null
      setReady(false)
      editor.destroy()
    }
  }, [container])

  return {
    ready,
    content: () => {
      const editor = editorRef.current
      if (!editor) return { html: optsRef.current.html, design: optsRef.current.design ?? {} }
      return {
        html: String(editor.runCommand(INLINE_HTML) ?? editor.getHtml()),
        design: editor.getProjectData() as unknown as Record<string, unknown>,
      }
    },
    insertText: (text: string) => {
      const editor = editorRef.current
      if (!editor) return
      // While a text is being edited, type the token at the cursor.
      if (editor.getEditing()) {
        editor.Canvas.getWindow()?.focus()
        editor.Canvas.getDocument()?.execCommand('insertText', false, text)
        return
      }
      const target = editor.getSelected() ?? editor.getWrapper()
      target?.append({ type: 'text', content: text })
    },
    insertHTML: (html: string) => {
      const editor = editorRef.current
      if (!editor) return
      const target = editor.getSelected() ?? editor.getWrapper()
      target?.append(html)
    },
  }
}
