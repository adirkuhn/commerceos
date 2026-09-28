import { useEffect, useRef, useState } from 'react'
import grapesjs, { type Editor } from 'grapesjs'
import { registry } from '../components/registry'
import { installRegistry } from './grapes/plugin'

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

/** Creates the GrapesJS editor once its mount points exist. Everything
 *  generic GrapesJS ships with (HTML import, code view, default blocks,
 *  style manager, storage) is off: merchants only get CommerceOS components. */
export function useGrapesEditor(mounts: () => Mounts): Editor | undefined {
  const [editor, setEditor] = useState<Editor>()
  const created = useRef(false)

  useEffect(() => {
    const m = mounts()
    if (created.current || !m.canvas || !m.blocks || !m.layers) return
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
    installRegistry(e, registry)
    setEditor(e)
    return () => {
      e.destroy()
      created.current = false
    }
  }, [mounts])

  return editor
}
