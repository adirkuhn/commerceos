import { useCallback, useEffect, useMemo, useRef, useState, type ReactElement } from 'react'
import { QueryClient } from '@tanstack/react-query'
import 'grapesjs/dist/css/grapes.min.css'
import baseCSS from '../theme/base.css?inline'
import { StorefrontAPI } from '../api/storefront'
import { defaultSettings } from '../core/settings'
import type { EditorAPI } from './api'
import { AssetStore } from './assetStore'
import { PreviewStore } from './previewStore'
import { ValueStore } from './valueStore'
import { PreviewBar } from './PreviewBar'
import { usePreviewData } from './usePreviewData'
import { CanvasProviders } from './CanvasProviders'
import { syncCanvasStyles } from './grapes/canvasStyles'
import { Inspector } from './inspector/Inspector'
import { EditorServicesProvider, type EditorServices } from './services'
import { Toolbar } from './Toolbar'
import { useDraft } from './useDraft'
import { DEVICES, useGrapesEditor, type DeviceId } from './useGrapesEditor'

/** The CommerceOS theme editor: toolbar; components + layers; the GrapesJS
 *  canvas; and the inspector (component settings/styles or theme settings). */
export function EditorApp({ api, apiBase }: { api: EditorAPI; apiBase: string }) {
  const canvasRef = useRef<HTMLDivElement>(null)
  const blocksRef = useRef<HTMLDivElement>(null)
  const layersRef = useRef<HTMLDivElement>(null)
  const [assets] = useState(() => new AssetStore())
  const [preview] = useState(() => new PreviewStore())
  const [settingsStore] = useState(() => new ValueStore(defaultSettings()))
  const [queryClient] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 60_000 } } }))
  const [channelId, setChannelId] = useState('')

  const services: EditorServices | undefined = useMemo(() => channelId
    ? { api, assets, preview, settings: settingsStore, queryClient, channelId, store: new StorefrontAPI({ baseURL: apiBase, storeKey: channelId }) }
    : undefined, [api, apiBase, assets, preview, settingsStore, queryClient, channelId])

  const wrap = useCallback((el: ReactElement) => (services ? <CanvasProviders services={services}>{el}</CanvasProviders> : el), [services])
  const mounts = useCallback(() => ({ canvas: canvasRef.current, blocks: blocksRef.current, layers: layersRef.current }), [])
  const editor = useGrapesEditor(mounts, wrap)
  const { draft, doc, setDoc, dirty, status, save, publish, preview: openPreview, saveSettings, savingSettings } = useDraft(api, editor, assets)
  const [device, setDevice] = useState<DeviceId>('desktop')
  const previewData = usePreviewData(services?.store, preview, doc)
  const settings = useMemo(() => draft?.version.settings ?? defaultSettings(), [draft?.version.settings])

  useEffect(() => settingsStore.set(settings), [settingsStore, settings])

  useEffect(() => {
    if (draft) setChannelId(draft.sales_channel_id)
  }, [draft])

  useEffect(() => {
    editor?.setDevice(DEVICES.find(d => d.id === device)!.name)
  }, [editor, device])

  useEffect(() => {
    if (!editor) return
    const sync = () => syncCanvasStyles(editor, baseCSS, settings, assets.snapshot())
    editor.on('load update component:update', sync)
    const unsubscribe = assets.subscribe(sync)
    sync()
    return () => {
      editor.off('load update component:update', sync)
      unsubscribe()
    }
  }, [editor, settings, assets])

  return (
    <EditorServicesProvider value={services ?? null}>
      <div className="flex h-screen flex-col bg-neutral-100 text-neutral-900">
        <Toolbar
          themeName={draft?.theme.name ?? 'Theme'} documents={draft?.documents ?? []} doc={doc} onDoc={setDoc}
          editor={editor} device={device} onDevice={setDevice} dirty={dirty} status={status}
          onSave={() => void save()} onPreview={() => void openPreview(preview.snapshot().product?.slug)} onPublish={() => void publish()}
        />
        {services && <PreviewBar kind={previewData.kind} error={previewData.error} onProduct={previewData.chooseProduct} onCollection={previewData.chooseCollection} />}
        <div className="flex min-h-0 flex-1">
          <aside className="w-64 shrink-0 overflow-y-auto border-r border-neutral-200 bg-white" aria-label="Components and layers">
            <h2 className="px-3 pt-3 text-xs font-semibold uppercase tracking-wide text-neutral-500">Components</h2>
            <div ref={blocksRef} />
            <h2 className="px-3 pt-3 text-xs font-semibold uppercase tracking-wide text-neutral-500">Layers</h2>
            <div ref={layersRef} />
          </aside>
          <main className="min-w-0 flex-1" aria-label="Canvas"><div ref={canvasRef} className="h-full" /></main>
          <aside className="w-80 shrink-0 overflow-y-auto border-l border-neutral-200 bg-white p-4" aria-label="Inspector">
            {editor && services && (
              <Inspector editor={editor} device={device} settings={settings} onSaveSettings={s => void saveSettings(s)} savingSettings={savingSettings} />
            )}
          </aside>
        </div>
      </div>
    </EditorServicesProvider>
  )
}
