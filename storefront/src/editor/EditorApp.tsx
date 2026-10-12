import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore, type ReactElement } from 'react'
import { useTranslation } from 'react-i18next'
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
import { ContentLanguageSelect } from './ContentLanguageSelect'
import { storeQuery } from '../runtime/page/queries'
import { storeLanguages, type StoreLanguages } from '../runtime/storeLocale'
import type { Locale } from '../i18n/locale'
import { useDraft } from './useDraft'
import { useEditorLabels } from './i18n/useEditorLabels'
import { DEVICES, useGrapesEditor, type DeviceId } from './useGrapesEditor'
import { useEditorLook } from './useEditorLook'
import { DEFAULT_EDITOR_APPEARANCE, type ThemePreference } from './editorLook'
import { EditorSidePanel, type SidePanelTab } from './EditorSidePanel'
import './grapes/editorChrome.css'

/** The CommerceOS theme editor: toolbar; components + layers; the GrapesJS
 *  canvas; and the inspector (component settings/styles or theme settings). */
export function EditorApp({ api, apiBase, themePreference = 'system' }: { api: EditorAPI; apiBase: string; themePreference?: ThemePreference }) {
  const canvasRef = useRef<HTMLDivElement>(null)
  const blocksRef = useRef<HTMLDivElement>(null)
  const layersRef = useRef<HTMLDivElement>(null)
  const { t } = useTranslation('editor')
  const labels = useEditorLabels()
  const [assets] = useState(() => new AssetStore())
  const [preview] = useState(() => new PreviewStore())
  const [settingsStore] = useState(() => new ValueStore(defaultSettings()))
  const [queryClient] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 60_000 } } }))
  const [channelId, setChannelId] = useState('')
  const [contentLocale] = useState(() => new ValueStore<Locale | ''>(''))
  const textsIn = useSyncExternalStore(contentLocale.subscribe, contentLocale.snapshot)
  const [languages, setLanguages] = useState<StoreLanguages>()

  const services: EditorServices | undefined = useMemo(() => channelId
    ? { api, assets, preview, settings: settingsStore, queryClient, channelId, contentLocale, store: new StorefrontAPI({ baseURL: apiBase, storeKey: channelId }) }
    : undefined, [api, apiBase, assets, preview, settingsStore, queryClient, channelId, contentLocale])

  const wrap = useCallback((el: ReactElement) => (services ? <CanvasProviders services={services}>{el}</CanvasProviders> : el), [services])
  const mounts = useCallback(() => ({ canvas: canvasRef.current, blocks: blocksRef.current, layers: layersRef.current }), [])
  // GrapesJS starts once the draft is open: by then the editor speaks the
  // session's language, which its blocks and layer names are created in.
  const [sessionReady, setSessionReady] = useState(false)
  const editor = useGrapesEditor(mounts, wrap, labels, sessionReady)
  const { draft, doc, setDoc, dirty, status, save, publish, preview: openPreview, saveSettings, savingSettings, texts, setTexts } = useDraft(api, editor, assets)
  const [device, setDevice] = useState<DeviceId>('desktop')
  const [sideTab, setSideTab] = useState<SidePanelTab>('add')
  // A backend from before appearances sends none: the default look then.
  useEditorLook(themePreference, draft ? (draft.appearance ?? DEFAULT_EDITOR_APPEARANCE) : undefined)
  const previewData = usePreviewData(services?.store, preview, doc)
  const settings = useMemo(() => draft?.version.settings ?? defaultSettings(), [draft?.version.settings])

  useEffect(() => settingsStore.set(settings), [settingsStore, settings])

  // The languages the store offers decide which texts can be translated.
  useEffect(() => {
    if (!services) return
    let cancelled = false
    queryClient.fetchQuery(storeQuery(services.store))
      .then(store => { if (!cancelled) setLanguages(storeLanguages(store)) })
      .catch((e: unknown) => console.warn('editor: store languages unavailable', e))
    return () => { cancelled = true }
  }, [services, queryClient])

  useEffect(() => {
    if (!draft) return
    setChannelId(draft.sales_channel_id)
    setSessionReady(true)
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
      <div className="flex h-screen flex-col bg-ed-canvas text-ed-ink">
        <Toolbar
          themeName={draft?.theme.name ?? t('layout.themeFallback')} documents={draft?.documents ?? []} doc={doc} onDoc={setDoc}
          editor={editor} device={device} onDevice={setDevice} dirty={dirty} status={status}
          onSave={() => void save()} onPreview={() => void openPreview(preview.snapshot().product?.slug)} onPublish={() => void publish()}
        >
          <ContentLanguageSelect languages={languages} value={textsIn} onChange={l => contentLocale.set(l)} />
        </Toolbar>
        {services && <PreviewBar kind={previewData.kind} error={previewData.error} onProduct={previewData.chooseProduct} onCollection={previewData.chooseCollection} />}
        <div className="flex min-h-0 flex-1">
          <EditorSidePanel tab={sideTab} onTab={setSideTab} blocksRef={blocksRef} layersRef={layersRef} />
          <main className="cos-editor-canvas min-w-0 flex-1" aria-label={t('layout.canvas')}><div ref={canvasRef} className="h-full" /></main>
          <aside className="w-80 shrink-0 overflow-y-auto border-l border-ed-line bg-ed-raised p-4" aria-label={t('layout.inspector')}>
            {editor && services && (
              <Inspector editor={editor} device={device} settings={settings} onSaveSettings={s => void saveSettings(s)} savingSettings={savingSettings}
                page={{ kind: doc.kind, texts, onChange: setTexts }} />
            )}
          </aside>
        </div>
      </div>
    </EditorServicesProvider>
  )
}
