import { useCallback, useEffect, useRef, useState } from 'react'
import 'grapesjs/dist/css/grapes.min.css'
import baseCSS from '../theme/base.css?inline'
import { defaultSettings } from '../core/settings'
import type { EditorAPI } from './api'
import { syncCanvasStyles } from './grapes/canvasStyles'
import { Toolbar } from './Toolbar'
import { useDraft } from './useDraft'
import { DEVICES, useGrapesEditor, type DeviceId } from './useGrapesEditor'

/** The CommerceOS theme editor: toolbar, component library + layers on the
 *  left, the GrapesJS canvas in the middle. (Inspector: Phase 3.) */
export function EditorApp({ api }: { api: EditorAPI }) {
  const canvasRef = useRef<HTMLDivElement>(null)
  const blocksRef = useRef<HTMLDivElement>(null)
  const layersRef = useRef<HTMLDivElement>(null)
  const mounts = useCallback(() => ({ canvas: canvasRef.current, blocks: blocksRef.current, layers: layersRef.current }), [])
  const editor = useGrapesEditor(mounts)
  const { draft, doc, setDoc, dirty, status, save, publish, preview } = useDraft(api, editor)
  const [device, setDevice] = useState<DeviceId>('desktop')

  useEffect(() => {
    editor?.setDevice(DEVICES.find(d => d.id === device)!.name)
  }, [editor, device])

  useEffect(() => {
    if (!editor) return
    const settings = draft?.version.settings ?? defaultSettings()
    const sync = () => syncCanvasStyles(editor, baseCSS, settings)
    editor.on('load update component:update', sync)
    sync()
    return () => {
      editor.off('load update component:update', sync)
    }
  }, [editor, draft?.version.settings])

  return (
    <div className="flex h-screen flex-col bg-neutral-100 text-neutral-900">
      <Toolbar
        themeName={draft?.theme.name ?? 'Theme'} documents={draft?.documents ?? []} doc={doc} onDoc={setDoc}
        editor={editor} device={device} onDevice={setDevice} dirty={dirty} status={status}
        onSave={() => void save()} onPreview={() => void preview()} onPublish={() => void publish()}
      />
      <div className="flex min-h-0 flex-1">
        <aside className="w-64 shrink-0 overflow-y-auto border-r border-neutral-200 bg-white" aria-label="Components and layers">
          <h2 className="px-3 pt-3 text-xs font-semibold uppercase tracking-wide text-neutral-500">Components</h2>
          <div ref={blocksRef} />
          <h2 className="px-3 pt-3 text-xs font-semibold uppercase tracking-wide text-neutral-500">Layers</h2>
          <div ref={layersRef} />
        </aside>
        <main className="min-w-0 flex-1" aria-label="Canvas"><div ref={canvasRef} className="h-full" /></main>
      </div>
    </div>
  )
}
