import { useState, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { X, ImagePlus } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useProductMedia } from './hooks/useProductMedia'
import type { Media } from './types'
import { apiErrorText } from '../api/apiErrorText'

// ── Media section ─────────────────────────────────────────────────────────────

const ACCEPTED_IMAGE_TYPES = 'image/jpeg,image/png,image/webp,image/gif'

export function MediaSection({ productId }: { productId: string }) {
  const { token } = useAuth()
  const { t } = useTranslation(['pms', 'common'])
  const { media, loading, error, upload, update, remove } = useProductMedia(token ?? '', productId)
  const [uploading, setUploading] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [titleDraft, setTitleDraft] = useState('')
  const [titleSaving, setTitleSaving] = useState(false)
  const [titleError, setTitleError] = useState('')
  const [dragging, setDragging] = useState(false)

  const featured = media[0] ?? null
  const rest = media.slice(1)

  // Uploaded sequentially, not in parallel -- each upload's success handler
  // appends to `media` via a functional setState update, but running them
  // one at a time keeps upload order == drop/selection order and avoids
  // firing a burst of simultaneous multipart requests for a large batch.
  async function handleFiles(files: FileList | Iterable<File> | null) {
    if (!files) return
    const list = Array.from(files)
    if (list.length === 0) return
    setUploading(true)
    setUploadError(null)
    try {
      for (const file of list) {
        await upload(file)
      }
    } catch (e) {
      setUploadError(apiErrorText(e) ?? t('media.uploadFailed'))
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  function handleDrop(e: React.DragEvent<HTMLDivElement>) {
    e.preventDefault()
    setDragging(false)
    handleFiles(e.dataTransfer.files)
  }

  function openTitleEditor(m: Media) {
    setEditingId(m.id)
    setTitleDraft(m.alt_text)
    setTitleError('')
  }

  async function handleSaveTitle() {
    if (!editingId) return
    setTitleSaving(true)
    setTitleError('')
    try {
      await update(editingId, titleDraft)
      setEditingId(null)
    } catch (e) {
      setTitleError(apiErrorText(e) ?? t('media.titleFailed'))
    } finally {
      setTitleSaving(false)
    }
  }

  return (
    <div className="space-y-3">
      <input
        ref={fileInputRef}
        type="file"
        accept={ACCEPTED_IMAGE_TYPES}
        multiple
        className="hidden"
        onChange={e => handleFiles(e.target.files)}
      />
      <div
        data-testid="media-dropzone"
        onDragOver={e => { e.preventDefault(); setDragging(true) }}
        onDragLeave={() => setDragging(false)}
        onDrop={handleDrop}
        className={clsx(
          'grid grid-cols-2 gap-3 rounded-xl transition-colors',
          dragging && 'ring-2 ring-focus ring-offset-2 ring-offset-transparent',
        )}
      >
        <div className="group relative aspect-square rounded-xl border border-line overflow-hidden flex items-center justify-center text-2xs text-ink-4">
          {featured ? (
            <>
              <button
                type="button"
                onClick={() => openTitleEditor(featured)}
                aria-label={t('media.editTitle')}
                className="w-full h-full cursor-pointer"
              >
                <img src={featured.url} alt={featured.alt_text} className="w-full h-full object-cover" />
              </button>
              <button
                type="button"
                onClick={() => remove(featured.id)}
                aria-label={t('media.removeImage')}
                className="absolute top-1.5 right-1.5 p-1 rounded-full bg-black/60 text-white opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity"
              >
                <X size={12} />
              </button>
            </>
          ) : (
            <span>{t('media.featured')}</span>
          )}
        </div>
        <div className="grid grid-cols-2 gap-2 content-start">
          {rest.map(m => (
            <div key={m.id} className="group relative aspect-square rounded-lg border border-line overflow-hidden">
              <button
                type="button"
                onClick={() => openTitleEditor(m)}
                aria-label={t('media.editTitle')}
                className="w-full h-full cursor-pointer"
              >
                <img src={m.url} alt={m.alt_text} className="w-full h-full object-cover" />
              </button>
              <button
                type="button"
                onClick={() => remove(m.id)}
                aria-label={t('media.removeImage')}
                className="absolute top-1 right-1 p-0.5 rounded-full bg-black/60 text-white opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity"
              >
                <X size={10} />
              </button>
            </div>
          ))}
          <button
            type="button"
            disabled={uploading}
            onClick={() => fileInputRef.current?.click()}
            aria-label={t('media.addImage')}
            title={t('media.addImage')}
            className="aspect-square rounded-lg border-2 border-dashed border-line flex items-center justify-center text-ink-4 hover:text-ink-3 disabled:opacity-50"
          >
            <ImagePlus size={16} />
          </button>
        </div>
      </div>
      {editingId && (
        <div className="p-2.5 rounded-lg border border-line space-y-2">
          <label htmlFor="media-title" className="block text-2xs font-medium text-ink-4 caps-label">
            {t('media.imageTitle')}
          </label>
          <input
            id="media-title"
            type="text"
            value={titleDraft}
            onChange={e => setTitleDraft(e.target.value)}
            placeholder={t('media.describe')}
            className="w-full px-2.5 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
          />
          {titleError && <p className="text-2xs text-danger">{titleError}</p>}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setEditingId(null)}
              className="text-2xs text-ink-4 hover:text-ink-2 transition-colors"
            >
              {t('common:actions.cancel')}
            </button>
            <button
              type="button"
              disabled={titleSaving}
              onClick={handleSaveTitle}
              className="text-2xs font-medium px-2.5 py-1 rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
            >
              {titleSaving ? t('common:actions.saving') : t('media.saveTitle')}
            </button>
          </div>
        </div>
      )}
      {loading && <p className="text-2xs text-ink-4">{t('common:status.loading')}</p>}
      {(uploadError || error) && (
        <p className="text-2xs text-danger">{uploadError ?? error}</p>
      )}
    </div>
  )
}
