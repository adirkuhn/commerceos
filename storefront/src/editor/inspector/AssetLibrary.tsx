import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { editorErrorText } from '../editorErrorText'
import type { Asset } from '../api'
import { useEditorServices } from '../services'

interface Props {
  onPick: (asset: Asset) => void
  onClose: () => void
}

/** The tenant's theme media library: upload, pick, remove images. */
export function AssetLibrary({ onPick, onClose }: Props) {
  const { api, assets: store } = useEditorServices()
  const [items, setItems] = useState<Asset[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [uploading, setUploading] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)
  const { t, i18n } = useTranslation('editor')

  useEffect(() => {
    api.listAssets().then(r => setItems(r.items), e => setError(editorErrorText(e, i18n, t('media.loadFailed')))).finally(() => setLoading(false))
  }, [api])

  const remember = (a: Asset) => store.merge({ [a.id]: { url: a.url, width: a.width, height: a.height, alt: a.alt } })

  const upload = async (files: FileList | null) => {
    const file = files?.[0]
    if (!file) return
    setUploading(true)
    setError('')
    try {
      const a = await api.uploadAsset(file)
      remember(a)
      setItems(list => [a, ...list])
    } catch (e) {
      setError(editorErrorText(e, i18n, t('media.uploadFailed')))
    } finally {
      setUploading(false)
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  const remove = async (a: Asset) => {
    try {
      await api.deleteAsset(a.id)
      setItems(list => list.filter(i => i.id !== a.id))
    } catch (e) {
      setError(editorErrorText(e, i18n, t('media.removeFailed')))
    }
  }

  return (
    <div role="dialog" aria-modal="true" aria-label={t('media.title')} className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      onKeyDown={e => e.key === 'Escape' && onClose()}>
      <div className="flex max-h-[80vh] w-[42rem] max-w-[95vw] flex-col rounded-lg bg-white shadow-xl">
        <header className="flex items-center gap-2 border-b px-4 py-3">
          <h2 className="font-semibold">{t('media.title')}</h2>
          <label className="ml-auto cursor-pointer rounded-md bg-blue-700 px-3 py-1.5 text-sm font-medium text-white">
            {uploading ? t('media.uploading') : t('media.upload')}
            <input ref={fileRef} type="file" accept="image/png,image/jpeg,image/webp,image/gif,image/x-icon" className="sr-only"
              disabled={uploading} onChange={e => void upload(e.target.files)} />
          </label>
          <button type="button" className="rounded-md px-2 py-1.5 text-sm" onClick={onClose}>{t('media.close')}</button>
        </header>
        {error && <p role="alert" className="px-4 pt-3 text-sm text-red-700">{error}</p>}
        <div className="grid grid-cols-4 gap-3 overflow-y-auto p-4">
          {loading && <p className="col-span-4 text-sm text-neutral-500">{t('media.loading')}</p>}
          {!loading && !items.length && <p className="col-span-4 text-sm text-neutral-500">{t('media.empty')}</p>}
          {items.map(a => (
            <div key={a.id} className="group relative">
              <button type="button" className="block aspect-square w-full overflow-hidden rounded-md border hover:ring-2 hover:ring-blue-600"
                onClick={() => { remember(a); onPick(a) }} aria-label={t('media.use', { name: a.alt || a.filename })}>
                <img src={a.url} alt="" className="h-full w-full object-cover" />
              </button>
              <button type="button" className="absolute right-1 top-1 hidden rounded bg-white/90 px-1.5 text-xs text-red-700 group-hover:block"
                onClick={() => void remove(a)} aria-label={t('media.removeNamed', { name: a.alt || a.filename })}>{t('media.remove')}</button>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
