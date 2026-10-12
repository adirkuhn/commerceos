import { useEffect, useRef, useState, type RefObject } from 'react'
import grapesjs, { type Editor, type ProjectData } from 'grapesjs'
import presetNewsletter from 'grapesjs-preset-newsletter'
import nl from 'grapesjs/locale/nl.mjs'
import pt from 'grapesjs/locale/pt.mjs'
import 'grapesjs/dist/css/grapes.min.css'
import './emailEditor.css'
import type { EditorBlock } from './editorBlocks'
import { emailStyleSectors, type SectorLabels } from './styleSectors'

export type EditorDevice = 'desktop' | 'mobile'

const DEVICE_NAMES: Record<EditorDevice, string> = { desktop: 'Desktop', mobile: 'Mobile portrait' }

export interface EmailEditorOptions {
  // design reopens a saved project; without one, html seeds the canvas.
  design?: Record<string, unknown> | null
  html: string
  blocks: EditorBlock[]
  blockCategory: string
  locale: string
  // Our own panels the editor renders its block list, style settings and
  // layer tree into (instead of GrapesJS's built-in panel bar).
  panels: {
    blocks: RefObject<HTMLDivElement>
    styles: RefObject<HTMLDivElement>
    layers: RefObject<HTMLDivElement>
  }
  sectorLabels: SectorLabels
  onSelect?: () => void
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
  undo: () => void
  redo: () => void
  device: EditorDevice
  setDevice: (device: EditorDevice) => void
  showCode: () => void
}

const INLINE_HTML = 'gjs-get-inlined-html'
const EXPORT_CODE = 'export-template'

// useEmailEditor mounts GrapesJS with the newsletter preset (email-safe
// blocks, inline CSS on export) plus the CommerceOS blocks into container,
// with its blocks, styles and layers drawn into the page's own panels.
export function useEmailEditor(container: React.RefObject<HTMLDivElement>, opts: EmailEditorOptions): EmailEditor {
  const editorRef = useRef<Editor | null>(null)
  const optsRef = useRef(opts)
  optsRef.current = opts
  const [ready, setReady] = useState(false)
  const [device, setDeviceState] = useState<EditorDevice>('desktop')

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
      pluginsOpts: {
        [presetNewsletter as unknown as string]: {
          inlineCss: true, useCustomTheme: false, updateStyleManager: false, showBlocksOnLoad: false, showStylesOnChange: false,
        },
      },
      // The email shows as a page on the well (emails are ~600px wide).
      deviceManager: {
        devices: [
          { id: 'desktop', name: DEVICE_NAMES.desktop, width: '760px' },
          { id: 'mobilePortrait', name: DEVICE_NAMES.mobile, width: '375px', widthMedia: '480px' },
        ],
      },
      blockManager: { appendTo: o.panels.blocks.current ?? undefined },
      styleManager: { appendTo: o.panels.styles.current ?? undefined, sectors: emailStyleSectors(o.sectorLabels) },
      layerManager: { appendTo: o.panels.layers.current ?? undefined },
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
    editor.on('load', () => {
      // The preset builds GrapesJS's own panel bar; our page has its own.
      editor.Panels.getPanels().reset([])
      setReady(true)
    })
    editor.on('component:selected', () => optsRef.current.onSelect?.())
    return () => {
      editorRef.current = null
      setReady(false)
      setDeviceState('desktop')
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
    undo: () => { editorRef.current?.UndoManager.undo() },
    redo: () => { editorRef.current?.UndoManager.redo() },
    device,
    setDevice: (next: EditorDevice) => {
      editorRef.current?.setDevice(DEVICE_NAMES[next])
      setDeviceState(next)
    },
    showCode: () => { editorRef.current?.runCommand(EXPORT_CODE) },
  }
}
