import { useState, useEffect } from 'react'
import { Trash2 } from 'lucide-react'
import clsx from 'clsx'
import * as api from './api'
import { useCategories } from '../pms/hooks/useCategories'
import { useExternalCategoryLinks } from './hooks/useExternalCategoryLinks'
import type { WooCategory, ExternalCategoryLink } from './types'

export function WooCategoriesTab({ token, connectionId }: { token: string; connectionId: string }) {
  const { categories } = useCategories(token)
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
      .catch(err => { if (!cancelled) setWooCategoriesError(err instanceof Error ? err.message : 'Failed to load WooCommerce categories') })
      .finally(() => { if (!cancelled) setWooCategoriesLoading(false) })
    return () => { cancelled = true }
  }, [token, connectionId])

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
      setRowError(err instanceof Error ? err.message : 'Failed to update category link')
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
      setRowError(err instanceof Error ? err.message : 'Failed to add category link')
    }
  }

  async function handleRemove(id: string) {
    setRowError('')
    try {
      await removeLink(id)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to delete category link')
    }
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="max-w-3xl space-y-4">
        <div>
          <h2 className="text-sm font-semibold text-ink mb-1">WooCommerce categories</h2>
          <p className="text-xs text-ink-4">
            A CommerceOS category is auto-matched to a WooCommerce category with the same name the first time a
            product in it is pushed. Repoint or remove a link, or add one manually for a category that hasn't
            matched automatically.
          </p>
        </div>

        {linksError && <p className="text-xs text-danger">{linksError}</p>}
        {wooCategoriesError && <p className="text-xs text-danger">{wooCategoriesError}</p>}
        {rowError && <p className="text-xs text-danger">{rowError}</p>}

        <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
          {links.length === 0 && <p className="text-xs text-ink-4">No category links yet — push a product to auto-match, or add one below.</p>}
          {links.length > 0 && (
            <div className="space-y-1.5">
              {links.map(link => (
                <div key={link.id} className="flex items-center gap-2 text-xs text-ink-2">
                  <span className="flex-1 truncate">{categoryLabel(link.category_id)}</span>
                  <select
                    aria-label="Linked Woo category"
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
                    <Trash2 size={11} /> Remove link
                  </button>
                </div>
              ))}
            </div>
          )}
          <form onSubmit={handleAdd} className="pt-3 border-t border-line flex items-end gap-2">
            <div>
              <label htmlFor="commerceos-category" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                Category
              </label>
              <select
                id="commerceos-category"
                aria-label="Category"
                required
                value={newCategoryId}
                onChange={e => setNewCategoryId(e.target.value)}
                className={clsx(inputClass, 'w-40')}
              >
                <option value="" disabled>Select…</option>
                {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </div>
            <div>
              <label htmlFor="woo-category" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                Woo category
              </label>
              <select
                id="woo-category"
                aria-label="Woo category"
                required
                disabled={wooCategoriesLoading || wooCategories.length === 0}
                value={newExternalId}
                onChange={e => setNewExternalId(e.target.value)}
                className={clsx(inputClass, 'w-32 disabled:opacity-40')}
              >
                <option value="" disabled>{wooCategoriesLoading ? 'Loading…' : 'Select…'}</option>
                {wooCategories.map(c => <option key={c.external_id} value={c.external_id}>{c.name}</option>)}
              </select>
            </div>
            <button
              type="submit"
              className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
            >
              Add link
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
