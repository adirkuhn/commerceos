import { useEffect, useRef, useState, type ReactElement } from 'react'
import grapesjs, { type Editor } from 'grapesjs'
import { registry } from '../components/registry'
import { installRegistry } from './grapes/plugin'
import type { EditorLabels } from './i18n/labels'

export const DEVICES = [
  { id: 'desktop', name: 'Desktop', width: '' },
  { id: 'tablet', name: 'Tablet', width: '768px', widthMedia: '1023px' },
  { id: 'mobile', name: 'Mobile', width: '375px', widthMedia: '767px' },
] as const

export type DeviceId = (typeof DEVICES)[number]['id']

interface Mounts {
  canvas: HTMLElement | null
  blocks: HTMLElement | null
  layers: HTMLElement | null
}

/** Creates the GrapesJS editor once its mount points exist and `ready`
 *  (the edit session is known, so blocks get their final language). Everything
 *  generic GrapesJS ships with (HTML import, code view, default blocks,
 *  style manager, storage) is off: merchants only get CommerceOS components. */
export function useGrapesEditor(mounts: () => Mounts, wrap: (el: ReactElement) => ReactElement, labels: EditorLabels, ready = true): Editor | undefined {
  const [editor, setEditor] = useState<Editor>()
  const created = useRef(false)
  // The wrapper changes once the draft (and so the channel) is known; the
  // canvas components always use the latest one.
  const wrapRef = useRef(wrap)
  wrapRef.current = wrap
  // Block and layer names, in the editor's language (fixed for the session).
  const labelsRef = useRef(labels)
  labelsRef.current = labels

  useEffect(() => {
    const m = mounts()
    if (!ready || created.current || !m.canvas || !m.blocks || !m.layers) return
    created.current = true
    const e = grapesjs.init({
      container: m.canvas,
      height: '100%',
      width: 'auto',
      fromElement: false,
      storageManager: false,
      panels: { defaults: [] },
      blockManager: { appendTo: m.blocks },
      layerManager: { appendTo: m.layers },
      styleManager: { sectors: [] },
      selectorManager: { componentFirst: true },
      deviceManager: { devices: DEVICES.map(d => ({ ...d })) },
      undoManager: { trackSelection: false },
      canvas: { styles: [], scripts: [] },
    })
    installRegistry(e, registry, { wrap: el => wrapRef.current(el), labels: labelsRef.current })
    setEditor(e)
    return () => {
      e.destroy()
      created.current = false
    }
  }, [mounts, ready])

  return editor
}
