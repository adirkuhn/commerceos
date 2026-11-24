import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { Layers, Plus, Trash2 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { RecordLoading, RecordNotFound } from '../components/common/RecordStatus'
import { useConfirm } from '../components/common/useConfirm'
import { CollectionEditPage } from '../collections/CollectionEditPage'
import { getCollection } from '../collections/api'
import { useCollections } from '../collections/hooks/useCollections'
import type { Collection } from '../collections/types'
import { NEW_RECORD, useRoutedEditor } from '../routing/useRoutedEditor'
import { useResource } from '../routing/useResource'
import { apiErrorText } from '../api/apiErrorText'
import { LIST_TOOLBAR_SINGLE } from '../components/common/listToolbar'

function describe(t: TFunction<'collections'>, c: Collection): string {
  if (c.type === 'manual') return t('list.handPicked')
  const match = c.match === 'any' ? t('match.any') : t('match.all')
  return t('list.conditions', { count: c.rules.length, match })
}

function EditRoute({ id, onSave, onClose, onSaved }: { id: string; onSave: Parameters<typeof CollectionEditPage>[0]['onSave']; onClose: () => void; onSaved: (id: string) => void }) {
  const { token } = useAuth()
  const { t } = useTranslation('collections')
  const { data, loading } = useResource(() => getCollection(token ?? '', id), [token, id])
  if (loading) return <RecordLoading />
  if (!data) return <RecordNotFound label={t('list.recordLabel')} onBack={onClose} />
  return <CollectionEditPage key={id} collection={data} onSave={onSave} onClose={onClose} onSaved={onSaved} />
}

export default function Collections() {
  const { token } = useAuth()
  const { t } = useTranslation(['collections', 'common'])
  const { collections, loading, error, save, remove } = useCollections(token ?? '')
  const { editingId, openNew, openEdit, close, showSaved } = useRoutedEditor('/collections')
  const { confirm, confirmDialog } = useConfirm()
  const [actionError, setActionError] = useState('')

  if (editingId === NEW_RECORD) return <CollectionEditPage collection={null} onSave={save} onClose={close} onSaved={showSaved} />
  if (editingId !== null) return <EditRoute id={editingId} onSave={save} onClose={close} onSaved={close} />

  async function handleDelete(c: Collection) {
    if (!(await confirm({ title: t('list.deleteTitle', { name: c.name }), message: t('list.deleteMessage'), confirmLabel: t('list.deleteConfirm'), destructive: true }))) return
    setActionError('')
    try {
      await remove(c.id)
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('list.deleteFailed'))
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className={clsx('flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0', LIST_TOOLBAR_SINGLE)}>
        <p className="text-xs text-ink-4">{t('list.intro')}</p>
        <button onClick={openNew} className="ml-auto flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover">
          <Plus size={13} /> {t('list.add')}
        </button>
      </div>
      {(error || actionError) && <p role="alert" className="px-5 py-2 text-xs text-danger">{error || actionError}</p>}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colName')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colProducts')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colUrl')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colStatus')}</th>
              <th className="px-3 py-2.5 w-10" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>}
            {!loading && !collections.length && <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">{t('list.empty')}</td></tr>}
            {collections.map(c => (
              <tr key={c.id} onClick={() => openEdit(c.id)} className="group hover:bg-hover/60 cursor-pointer">
                <td className="px-2 py-2.5"><div className="w-6 h-6 rounded bg-fill flex items-center justify-center"><Layers size={12} className="text-ink-4" /></div></td>
                <td className="px-3 py-2.5 font-medium text-ink">{c.name}</td>
                <td className="px-3 py-2.5 text-ink-3">{describe(t, c)}</td>
                <td className="px-3 py-2.5 text-ink-3 font-mono">{`/collections/${c.slug}`}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-sm text-2xs font-semibold', c.published ? 'bg-success/10 text-success' : 'bg-fill text-ink-4')}>
                    {c.published ? t('list.visible') : t('list.hidden')}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <button aria-label={t('list.deleteAria', { name: c.name })} onClick={e => { e.stopPropagation(); void handleDelete(c) }}
                    className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover opacity-0 group-hover:opacity-100 focus:opacity-100">
                    <Trash2 size={12} className="text-ink-3 hover:text-danger" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {confirmDialog}
    </div>
  )
}
