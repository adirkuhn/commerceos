import { useTranslation } from 'react-i18next'
import { useState, useEffect } from 'react'
import { Trash2 } from 'lucide-react'
import clsx from 'clsx'
import * as api from './api'
import type { WooTaxClass } from './types'
import { useTaxClasses } from '../tax/hooks/useTaxClasses'
import { useExternalTaxClassLinks } from '../tax/hooks/useExternalTaxClassLinks'
import type { ExternalTaxClassLink } from '../tax/types'
import { apiErrorText } from '../api/apiErrorText'

const WOO_PROVIDER = 'woocommerce'

export function WooTaxesTab({ token, connectionId }: { token: string; connectionId: string }) {
  const { classes } = useTaxClasses(token)
  const { t } = useTranslation(['integrations', 'common'])
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
      .catch(err => { if (!cancelled) setWooTaxesError(apiErrorText(err) ?? t('wooTaxes.loadFailed')) })
      .finally(() => { if (!cancelled) setWooTaxesLoading(false) })
    return () => { cancelled = true }
  }, [token, connectionId, t])

  async function handleRepoint(link: ExternalTaxClassLink, taxClassId: string) {
    setRowError('')
    try {
      await upsertLink({ provider: link.provider, external_slug: link.external_slug, tax_class_id: taxClassId })
    } catch (err) {
      setRowError(apiErrorText(err) ?? t('wooTaxes.updateFailed'))
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
      setRowError(apiErrorText(err) ?? t('wooTaxes.addFailed'))
    }
  }

  async function handleRemove(id: string) {
    setRowError('')
    try {
      await removeLink(id)
    } catch (err) {
      setRowError(apiErrorText(err) ?? t('wooTaxes.deleteFailed'))
    }
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="max-w-3xl space-y-4">
        <div>
          <h2 className="text-sm font-semibold text-ink mb-1">{t('wooTaxes.title')}</h2>
          <p className="text-xs text-ink-4">
            {t('wooTaxes.intro')}
          </p>
        </div>

        {linksError && <p className="text-xs text-danger">{linksError}</p>}
        {wooTaxesError && <p className="text-xs text-danger">{wooTaxesError}</p>}
        {rowError && <p className="text-xs text-danger">{rowError}</p>}

        <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
          {wooLinks.length === 0 && <p className="text-xs text-ink-4">{t('wooTaxes.empty')}</p>}
          {wooLinks.length > 0 && (
            <div className="space-y-1.5">
              {wooLinks.map(link => (
                <div key={link.id} className="flex items-center gap-2 text-xs text-ink-2">
                  <span className="flex-1 truncate">{link.external_slug || t('wooTaxes.standard')}</span>
                  <select
                    aria-label={t('wooTaxes.linkedClass')}
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
                    <Trash2 size={11} /> {t('removeLink')}
                  </button>
                </div>
              ))}
            </div>
          )}
          <form onSubmit={handleAdd} className="pt-3 border-t border-line flex items-end gap-2">
            <div>
              <label htmlFor="woo-tax-slug" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                {t('wooTaxes.wooClass')}
              </label>
              <select
                id="woo-tax-slug"
                aria-label={t('wooTaxes.wooClass')}
                required
                disabled={wooTaxesLoading || wooTaxes.length === 0}
                value={newSlug}
                onChange={e => setNewSlug(e.target.value)}
                className={clsx(inputClass, 'w-32 disabled:opacity-40')}
              >
                <option value="" disabled>{wooTaxesLoading ? t('common:status.loading') : t('select')}</option>
                {wooTaxes.map(tx => <option key={tx.slug} value={tx.slug}>{tx.label}</option>)}
              </select>
            </div>
            <div>
              <label htmlFor="woo-tax-class" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                {t('wooTaxes.taxClass')}
              </label>
              <select
                id="woo-tax-class"
                aria-label={t('wooTaxes.taxClass')}
                required
                value={newClassId}
                onChange={e => setNewClassId(e.target.value)}
                className={clsx(inputClass, 'w-40')}
              >
                <option value="" disabled>{t('select')}</option>
                {classes.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
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
