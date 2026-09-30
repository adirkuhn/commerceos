import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Editor } from 'grapesjs'
import type { DocumentKind, ThemeSettings } from '../core/document'
import { registry } from '../components/registry'
import { EditorApiError, type Draft, type EditorAPI } from './api'
import { fromGrapes, toGrapes } from './grapes/serializer'
import type { AssetStore } from './assetStore'
import { editorLocale } from './editorLocale'
import { documentSaveBody } from './saveBody'
import type { PageTexts } from './inspector/PageTextsForm'

export interface DocRef {
  kind: DocumentKind
  key: string
}

// Where a document can be seen on the storefront. Templates that need a
// product/collection/order open the home page until the editor has a
// "preview as" picker (Phase 3).
const TEMPLATE_PATHS: Record<string, string> = {
  cart: 'cart', search: 'search', checkout: 'checkout', login: 'login', account: 'account', wishlist: 'wishlist',
}

export function storePathFor(doc: DocRef, productSlug?: string): string {
  if (doc.kind === 'page') return `pages/${doc.key}`
  if (doc.kind === 'template' && doc.key === 'product' && productSlug) return `products/${encodeURIComponent(productSlug)}`
  return doc.kind === 'template' ? (TEMPLATE_PATHS[doc.key] ?? '') : ''
}

export type Status =
  | { kind: 'idle' }
  | { kind: 'busy'; label: string }
  | { kind: 'ok'; message: string }
  | { kind: 'error'; message: string }

/** Draft state for the editor: which document is open, the draft revision
 *  every write must quote, and load/save/publish/preview actions. */
export function useDraft(api: EditorAPI, editor: Editor | undefined, assets: AssetStore) {
  const [draft, setDraft] = useState<Draft>()
  const [revision, setRevision] = useState(0)
  const [doc, setDoc] = useState<DocRef>({ kind: 'template', key: 'home' })
  const [dirty, setDirty] = useState(false)
  // The document as last loaded or saved. Dirty means the canvas differs
  // from it -- GrapesJS's own "update" also fires for loading itself, and
  // undoing back to the saved state should read as clean again.
  const baseline = useRef('')
  // A message to show once the page (re)loads -- publishing reloads the
  // next draft, and that load must not wipe "Published version N".
  const afterLoad = useRef<Status>()
  const snapshot = () => (editor ? JSON.stringify(fromGrapes(editor.getWrapper()!)) : '')
  // The open document's own texts (title, SEO, their translations), saved
  // with its tree.
  const [texts, setTextsState] = useState<PageTexts>({ title: '', seo: {} })
  const textsRef = useRef(texts)
  const textsBaseline = useRef('')
  const textsChanged = () => JSON.stringify(textsRef.current) !== textsBaseline.current
  const canvasChanged = () => baseline.current !== '' && snapshot() !== baseline.current
  const setTexts = (next: PageTexts) => {
    textsRef.current = next
    setTextsState(next)
    setDirty(canvasChanged() || textsChanged())
  }
  const { t, i18n } = useTranslation('editor')
  const [status, setStatus] = useState<Status>({ kind: 'busy', label: t('status.loadingTheme') })

  const fail = (err: unknown) => {
    if (err instanceof EditorApiError && err.status === 409) {
      setStatus({ kind: 'error', message: t('status.conflict') })
    } else if (err instanceof EditorApiError && err.nodeId && editor) {
      const target = editor.getWrapper()?.find('*').find(c => c.get('cosId') === err.nodeId)
      if (target) editor.select(target)
      setStatus({ kind: 'error', message: t('status.cantSave', { message: err.message }) })
    } else {
      setStatus({ kind: 'error', message: err instanceof Error ? err.message : t('status.somethingWrong') })
    }
  }

  const [savingSettings, setSavingSettings] = useState(false)

  const open = useCallback(async () => {
    const d = await api.openDraft()
    // The editor speaks the language of the admin who opened it.
    const locale = editorLocale(d.locale)
    if (i18n.language !== locale) await i18n.changeLanguage(locale)
    document.documentElement.lang = locale
    assets.merge(d.assets)
    setDraft(d)
    setRevision(d.version.revision)
    return d
  }, [api, assets, i18n])

  useEffect(() => {
    open().then(() => setStatus({ kind: 'idle' }), fail)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  useEffect(() => {
    if (!editor || !draft) return
    let cancelled = false
    baseline.current = '' // nothing to compare against until this page loads
    setStatus({ kind: 'busy', label: t('status.loadingPage') })
    api.getDocument(doc.kind, doc.key).then(d => {
      if (cancelled) return
      assets.merge(d.assets)
      const loaded: PageTexts = { title: d.title, seo: d.seo ?? {}, ...(d.translations ? { translations: d.translations } : {}) }
      textsRef.current = loaded
      setTextsState(loaded)
      textsBaseline.current = JSON.stringify(loaded)
      editor.setComponents(toGrapes(d.root, registry) as never)
      editor.UndoManager.clear()
      baseline.current = snapshot()
      setDirty(false)
      setStatus(afterLoad.current ?? { kind: 'idle' })
      afterLoad.current = undefined
    }, fail)
    const markDirty = () => setDirty(canvasChanged() || textsChanged())
    editor.on('update', markDirty)
    return () => {
      cancelled = true
      editor.off('update', markDirty)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [api, editor, draft?.version.id, doc.kind, doc.key])

  const save = useCallback(async (): Promise<number | undefined> => {
    if (!editor) return undefined
    setStatus({ kind: 'busy', label: t('status.saving') })
    try {
      const root = fromGrapes(editor.getWrapper()!)
      const res = await api.saveDocument(doc.kind, doc.key, documentSaveBody(doc.kind, revision, root, textsRef.current))
      setRevision(res.revision)
      baseline.current = JSON.stringify(root)
      textsBaseline.current = JSON.stringify(textsRef.current)
      setDirty(false)
      setStatus({ kind: 'ok', message: t('status.saved') })
      return res.revision
    } catch (err) {
      fail(err)
      return undefined
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [api, editor, doc, revision])

  const saveIfDirty = async () => (dirty ? save() : revision)

  const publish = async () => {
    const rev = await saveIfDirty()
    if (rev === undefined) return
    setStatus({ kind: 'busy', label: t('status.publishing') })
    try {
      const v = await api.publish(rev)
      afterLoad.current = { kind: 'ok', message: t('status.published', { number: v.number }) }
      await open() // publishing sealed this draft; continue on the next one
    } catch (err) {
      fail(err)
    }
  }

  const preview = async (productSlug?: string) => {
    if (!draft || (await saveIfDirty()) === undefined) return
    try {
      const { token } = await api.createPreviewSession()
      window.open(`/store/${draft.sales_channel_id}/${storePathFor(doc, productSlug)}?preview=${encodeURIComponent(token)}`, '_blank', 'noopener')
    } catch (err) {
      fail(err)
    }
  }

  const saveSettings = async (settings: ThemeSettings) => {
    if (!draft) return
    setSavingSettings(true)
    try {
      const v = await api.saveSettings(revision, settings)
      setRevision(v.revision)
      setDraft(d => (d ? { ...d, version: { ...d.version, settings: v.settings, revision: v.revision } } : d))
      setStatus({ kind: 'ok', message: t('status.settingsSaved') })
    } catch (err) {
      fail(err)
    } finally {
      setSavingSettings(false)
    }
  }

  return { draft, doc, setDoc, dirty, status, save, publish, preview, saveSettings, savingSettings, texts, setTexts }
}
