import { useState, useEffect } from 'react'
import { Trash2 } from 'lucide-react'
import clsx from 'clsx'
import * as api from './api'
import type { WooTaxClass } from './types'
import { useTaxClasses } from '../tax/hooks/useTaxClasses'
import { useExternalTaxClassLinks } from '../tax/hooks/useExternalTaxClassLinks'
import type { ExternalTaxClassLink } from '../tax/types'

const WOO_PROVIDER = 'woocommerce'

export function WooTaxesTab({ token, connectionId }: { token: string; connectionId: string }) {
  const { classes } = useTaxClasses(token)
  const { links, error: linksError, upsert: upsertLink, remove: removeLink } = useExternalTaxClassLinks(token)
  const wooLinks = links.filter(l => l.provider === WOO_PROVIDER)

  const inputClass = 'px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'

  const [rowError, setRowError] = useState('')
  const [newSlug, setNewSlug] = useState('')
  const [newClassId, setNewClassId] = useState('')

  const [wooTaxes, setWooTaxes] = useState<WooTaxClass[]>([])
  const [wooTaxesLoading, setWooTaxesLoading] = useState(true)
  const [wooTaxesError, setWooTaxesError] = useState('')

  useEffect(() => {
    if (!token || !connectionId) return
    let cancelled = false
    setWooTaxesLoading(true)
    setWooTaxesError('')
    api.listWooTaxes(token, connectionId)
      .then(result => { if (!cancelled) setWooTaxes(result) })
      .catch(err => { if (!cancelled) setWooTaxesError(err instanceof Error ? err.message : 'Failed to load WooCommerce taxes') })
      .finally(() => { if (!cancelled) setWooTaxesLoading(false) })
    return () => { cancelled = true }
  }, [token, connectionId])

  async function handleRepoint(link: ExternalTaxClassLink, taxClassId: string) {
    setRowError('')
    try {
      await upsertLink({ provider: link.provider, external_slug: link.external_slug, tax_class_id: taxClassId })
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to update tax link')
    }
  }

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    setRowError('')
    try {
      await upsertLink({ provider: WOO_PROVIDER, external_slug: newSlug, tax_class_id: newClassId })
      setNewSlug('')
      setNewClassId('')
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to add tax link')
    }
  }

  async function handleRemove(id: string) {
    setRowError('')
    try {
      await removeLink(id)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to delete tax link')
    }
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="max-w-3xl space-y-4">
        <div>
          <h2 className="text-sm font-semibold text-ink mb-1">WooCommerce taxes</h2>
          <p className="text-xs text-ink-4">
            Every WooCommerce tax class seen on import shows up here, linked to a CommerceOS tax class. Repoint or remove a link,
            or add one manually for a Woo tax class that hasn't been imported yet.
          </p>
        </div>

        {linksError && <p className="text-xs text-danger">{linksError}</p>}
        {wooTaxesError && <p className="text-xs text-danger">{wooTaxesError}</p>}
        {rowError && <p className="text-xs text-danger">{rowError}</p>}

        <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
          {wooLinks.length === 0 && <p className="text-xs text-ink-4">No WooCommerce taxes yet — run an import to pull them in, or add one below.</p>}
          {wooLinks.length > 0 && (
            <div className="space-y-1.5">
              {wooLinks.map(link => (
                <div key={link.id} className="flex items-center gap-2 text-xs text-ink-2">
                  <span className="flex-1 truncate">{link.external_slug || 'Standard'}</span>
                  <select
                    aria-label="Linked tax class"
                    value={link.tax_class_id}
                    onChange={e => handleRepoint(link, e.target.value)}
                    className={clsx(inputClass, 'w-40')}
                  >
                    {classes.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
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
              <label htmlFor="woo-tax-slug" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                Woo tax class
              </label>
              <select
                id="woo-tax-slug"
                aria-label="Woo tax class"
                required
                disabled={wooTaxesLoading || wooTaxes.length === 0}
                value={newSlug}
                onChange={e => setNewSlug(e.target.value)}
                className={clsx(inputClass, 'w-32 disabled:opacity-40')}
              >
                <option value="" disabled>{wooTaxesLoading ? 'Loading…' : 'Select…'}</option>
                {wooTaxes.map(t => <option key={t.slug} value={t.slug}>{t.label}</option>)}
              </select>
            </div>
            <div>
              <label htmlFor="woo-tax-class" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                Tax class
              </label>
              <select
                id="woo-tax-class"
                aria-label="Tax class"
                required
                value={newClassId}
                onChange={e => setNewClassId(e.target.value)}
                className={clsx(inputClass, 'w-40')}
              >
                <option value="" disabled>Select…</option>
                {classes.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
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
