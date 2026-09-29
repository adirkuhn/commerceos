import { useTranslation } from 'react-i18next'
import { useState, useEffect } from 'react'
import { Trash2 } from 'lucide-react'
import clsx from 'clsx'
import * as api from './api'
import { useCategories } from '../pms/hooks/useCategories'
import { useExternalCategoryLinks } from './hooks/useExternalCategoryLinks'
import type { WooCategory, ExternalCategoryLink } from './types'

export function WooCategoriesTab({ token, connectionId }: { token: string; connectionId: string }) {
  const { categories } = useCategories(token)
  const { t } = useTranslation(['integrations', 'common'])
  const { links, error: linksError, upsert: upsertLink, remove: removeLink } = useExternalCategoryLinks(token, connectionId)

  const inputClass = 'px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'

  const [rowError, setRowError] = useState('')
  const [newCategoryId, setNewCategoryId] = useState('')
  const [newExternalId, setNewExternalId] = useState('')

  const [wooCategories, setWooCategories] = useState<WooCategory[]>([])
  const [wooCategoriesLoading, setWooCategoriesLoading] = useState(true)
  const [wooCategoriesError, setWooCategoriesError] = useState('')

  useEffect(() => {
    if (!token || !connectionId) return
    let cancelled = false
    setWooCategoriesLoading(true)
    setWooCategoriesError('')
    api.listWooCategories(token, connectionId)
      .then(result => { if (!cancelled) setWooCategories(result) })
      .catch(err => { if (!cancelled) setWooCategoriesError(err instanceof Error ? err.message : t('wooCategories.loadFailed')) })
      .finally(() => { if (!cancelled) setWooCategoriesLoading(false) })
    return () => { cancelled = true }
  }, [token, connectionId, t])

  function wooCategoryLabel(externalId: string): string {
    const match = wooCategories.find(c => c.external_id === externalId)
    return match ? match.name : externalId
  }
  function categoryLabel(categoryId: string): string {
    const match = categories.find(c => c.id === categoryId)
    return match ? match.name : categoryId
  }

  async function handleRepoint(link: ExternalCategoryLink, externalId: string) {
    setRowError('')
    try {
      await upsertLink(externalId, link.category_id)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : t('wooCategories.updateFailed'))
    }
  }

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    setRowError('')
    try {
      await upsertLink(newExternalId, newCategoryId)
      setNewCategoryId('')
      setNewExternalId('')
    } catch (err) {
      setRowError(err instanceof Error ? err.message : t('wooCategories.addFailed'))
    }
  }

  async function handleRemove(id: string) {
    setRowError('')
    try {
      await removeLink(id)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : t('wooCategories.deleteFailed'))
    }
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="max-w-3xl space-y-4">
        <div>
          <h2 className="text-sm font-semibold text-ink mb-1">{t('wooCategories.title')}</h2>
          <p className="text-xs text-ink-4">
            {t('wooCategories.intro')}
          </p>
        </div>

        {linksError && <p className="text-xs text-danger">{linksError}</p>}
        {wooCategoriesError && <p className="text-xs text-danger">{wooCategoriesError}</p>}
        {rowError && <p className="text-xs text-danger">{rowError}</p>}

        <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
          {links.length === 0 && <p className="text-xs text-ink-4">{t('wooCategories.empty')}</p>}
          {links.length > 0 && (
            <div className="space-y-1.5">
              {links.map(link => (
                <div key={link.id} className="flex items-center gap-2 text-xs text-ink-2">
                  <span className="flex-1 truncate">{categoryLabel(link.category_id)}</span>
                  <select
                    aria-label={t('wooCategories.linked')}
                    value={link.external_id}
                    onChange={e => handleRepoint(link, e.target.value)}
                    className={clsx(inputClass, 'w-40')}
                  >
                    <option value={link.external_id}>{wooCategoryLabel(link.external_id)}</option>
                    {wooCategories.filter(c => c.external_id !== link.external_id).map(c => (
                      <option key={c.external_id} value={c.external_id}>{c.name}</option>
                    ))}
                  </select>
                  <button
                    onClick={() => handleRemove(link.id)}
                    className="text-2xs text-ink-4 hover:text-danger transition-colors flex items-center gap-1"
                  >
                    <Trash2 size={11} /> {t('removeLink')}
                  </button>
                </div>
              ))}
            </div>
          )}
          <form onSubmit={handleAdd} className="pt-3 border-t border-line flex items-end gap-2">
            <div>
              <label htmlFor="commerceos-category" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                {t('wooCategories.category')}
              </label>
              <select
                id="commerceos-category"
                aria-label={t('wooCategories.category')}
                required
                value={newCategoryId}
                onChange={e => setNewCategoryId(e.target.value)}
                className={clsx(inputClass, 'w-40')}
              >
                <option value="" disabled>{t('select')}</option>
                {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </div>
            <div>
              <label htmlFor="woo-category" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                {t('wooCategories.wooCategory')}
              </label>
              <select
                id="woo-category"
                aria-label={t('wooCategories.wooCategory')}
                required
                disabled={wooCategoriesLoading || wooCategories.length === 0}
                value={newExternalId}
                onChange={e => setNewExternalId(e.target.value)}
                className={clsx(inputClass, 'w-32 disabled:opacity-40')}
              >
                <option value="" disabled>{wooCategoriesLoading ? t('common:status.loading') : t('select')}</option>
                {wooCategories.map(c => <option key={c.external_id} value={c.external_id}>{c.name}</option>)}
              </select>
            </div>
            <button
              type="submit"
              className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
            >
              {t('addLink')}
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
