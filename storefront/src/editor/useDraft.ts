import { useCallback, useEffect, useState } from 'react'
import type { Editor } from 'grapesjs'
import type { DocumentKind, ThemeSettings } from '../core/document'
import { registry } from '../components/registry'
import { EditorApiError, type Draft, type EditorAPI } from './api'
import { fromGrapes, toGrapes } from './grapes/serializer'
import type { AssetStore } from './assetStore'

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
  const [status, setStatus] = useState<Status>({ kind: 'busy', label: 'Loading theme…' })

  const fail = (err: unknown) => {
    if (err instanceof EditorApiError && err.status === 409) {
      setStatus({ kind: 'error', message: 'This draft was changed somewhere else. Reload the editor to get the latest version.' })
    } else if (err instanceof EditorApiError && err.nodeId && editor) {
      const target = editor.getWrapper()?.find('*').find(c => c.get('cosId') === err.nodeId)
      if (target) editor.select(target)
      setStatus({ kind: 'error', message: `Can't save: ${err.message}` })
    } else {
      setStatus({ kind: 'error', message: err instanceof Error ? err.message : 'Something went wrong' })
    }
  }

  const [savingSettings, setSavingSettings] = useState(false)

  const open = useCallback(async () => {
    const d = await api.openDraft()
    assets.merge(d.assets)
    setDraft(d)
    setRevision(d.version.revision)
    return d
  }, [api, assets])

  useEffect(() => {
    open().then(() => setStatus({ kind: 'idle' }), fail)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  useEffect(() => {
    if (!editor || !draft) return
    let cancelled = false
    setStatus({ kind: 'busy', label: 'Loading page…' })
    api.getDocument(doc.kind, doc.key).then(d => {
      if (cancelled) return
      assets.merge(d.assets)
      editor.setComponents(toGrapes(d.root, registry) as never)
      editor.UndoManager.clear()
      setDirty(false)
      setStatus({ kind: 'idle' })
    }, fail)
    const markDirty = () => setDirty(true)
    editor.on('update', markDirty)
    return () => {
      cancelled = true
      editor.off('update', markDirty)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [api, editor, draft?.version.id, doc.kind, doc.key])

  const save = useCallback(async (): Promise<number | undefined> => {
    if (!editor) return undefined
    setStatus({ kind: 'busy', label: 'Saving…' })
    try {
      const res = await api.saveDocument(doc.kind, doc.key, { expected_revision: revision, root: fromGrapes(editor.getWrapper()!) })
      setRevision(res.revision)
      setDirty(false)
      setStatus({ kind: 'ok', message: 'Saved' })
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
    setStatus({ kind: 'busy', label: 'Publishing…' })
    try {
      const v = await api.publish(rev)
      await open() // publishing sealed this draft; continue on the next one
      setStatus({ kind: 'ok', message: `Published version ${v.number}. It's live now.` })
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
      setStatus({ kind: 'ok', message: 'Theme settings saved' })
    } catch (err) {
      fail(err)
    } finally {
      setSavingSettings(false)
    }
  }

  return { draft, doc, setDoc, dirty, status, save, publish, preview, saveSettings, savingSettings }
}
