import { useEffect, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { ArrowRight, X } from 'lucide-react'
import { Dialog } from '../components/common/Dialog'
import { useConfirm } from '../components/common/useConfirm'
import { formatMoney } from '../format/money'
import * as api from './api'
import { useProductMerge } from './hooks/useProductMerge'
import type { MergePreviewSKU, Product, ProductMergeResult } from './types'

interface MergeProductDialogProps {
  token: string
  duplicate: { id: string; name: string }
  currency: string
  // The chosen existing product (lives in the URL), or null while choosing.
  targetId: string | null
  onChooseTarget: (id: string | null) => void
  onMerged: (result: ProductMergeResult) => void
  onClose: () => void
}

const inputClass = 'w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink outline-none focus:ring-1 focus:ring-focus'

// Merge this (duplicate) product into an existing one: pick the existing
// product, match each SKU, see stock and cost after, then confirm twice --
// typing the existing product's name, then a final prompt. Irreversible.
export function MergeProductDialog({ token, duplicate, currency, targetId, onChooseTarget, onMerged, onClose }: MergeProductDialogProps) {
  const { preview, mapping, complete, loading, error, match, merge } = useProductMerge(token, duplicate.id, targetId)
  const { confirm, confirmDialog } = useConfirm()
  const { t } = useTranslation(['pms', 'common'])
  const [typedName, setTypedName] = useState('')
  const [merging, setMerging] = useState(false)
  const [mergeError, setMergeError] = useState<string | null>(null)
  useEffect(() => setTypedName(''), [targetId])

  const targetName = preview?.target.name ?? ''
  const nameConfirmed = targetName !== '' && typedName.trim() === targetName
  const money = (v?: number | null) => (v == null ? t('merge.unknown') : formatMoney(v, currency))

  const handleMerge = async () => {
    if (!(await confirm({
      title: t('merge.confirmTitle', { duplicate: duplicate.name, target: targetName }),
      message: t('merge.confirmMessage', { duplicate: duplicate.name }),
      confirmLabel: t('merge.confirmButton'), destructive: true,
    }))) return
    setMerging(true)
    setMergeError(null)
    try {
      onMerged(await merge())
    } catch (e) {
      setMergeError(e instanceof Error ? e.message : t('merge.failed'))
    } finally {
      setMerging(false)
    }
  }

  return (
    <Dialog onClose={onClose} dismissible={!merging}>
      <div className="w-full max-w-2xl rounded-xl bg-raised border border-line shadow-xl">
        <div className="flex items-center justify-between px-4 py-3 border-b border-line">
          <h3 className="text-sm font-semibold text-ink">{t('merge.title', { name: duplicate.name })}</h3>
          <button type="button" onClick={onClose} aria-label={t('common:actions.close')} className="p-1 -m-1 rounded text-ink-4 hover:text-ink-2 hover:bg-hover">
            <X size={16} />
          </button>
        </div>
        <div className="p-4 space-y-4">
          <p className="text-2xs text-ink-3">
            {t('merge.intro', { name: duplicate.name })}
          </p>

          {!targetId ? (
            <TargetSearch token={token} excludeId={duplicate.id} onPick={onChooseTarget} />
          ) : (
            <>
              <div className="flex items-center justify-between">
                <p className="text-xs text-ink-2">
                  <Trans t={t} i18nKey="merge.into" values={{ name: targetName || '…' }} components={{ strong: <strong className="text-ink" /> }} />
                </p>
                <button type="button" onClick={() => onChooseTarget(null)} className="text-2xs text-accent hover:text-accent/80">
                  {t('merge.chooseAnother')}
                </button>
              </div>

              {preview && (
                <table className="w-full text-2xs">
                  <thead>
                    <tr className="text-left text-ink-4 uppercase tracking-wider">
                      <th className="py-1 font-medium">{t('merge.colSku')}</th>
                      <th className="py-1 font-medium">{t('merge.colStockCost')}</th>
                      <th className="py-1 font-medium">{t('merge.colInto')}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-line/60">
                    {preview.source.skus.map(s => (
                      <tr key={s.id}>
                        <td className="py-1.5 font-mono text-ink-2">{skuLabel(s)}</td>
                        <td className="py-1.5 text-ink-3">{s.on_hand} · {money(s.cost_price_minor_units)}</td>
                        <td className="py-1.5">
                          <select
                            aria-label={t('merge.skuInto', { sku: s.sku_code })} value={mapping[s.id] ?? ''}
                            onChange={e => match(s.id, e.target.value)} className={inputClass}
                          >
                            <option value="" disabled>{t('merge.chooseSku')}</option>
                            {preview.target.skus.map(ts => <option key={ts.id} value={ts.id}>{t('merge.option', { label: skuLabel(ts), n: ts.on_hand })}</option>)}
                          </select>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}

              {complete && preview && preview.targets.length > 0 && (
                <div className="rounded-lg border border-line p-2.5 space-y-1" aria-label={t('merge.after')}>
                  <p className="text-2xs font-medium text-ink-3">{t('merge.after')}</p>
                  {preview.targets.map(tg => {
                    const sku = preview.target.skus.find(s => s.id === tg.sku_id)
                    return (
                      <p key={tg.sku_id} className="text-2xs text-ink-2 flex items-center gap-1.5 flex-wrap">
                        <span className="font-mono">{sku ? skuLabel(sku) : tg.sku_id}</span>
                        <span className="text-ink-4">{t('merge.stock')}</span> {sku?.on_hand ?? 0} <ArrowRight size={10} /> <strong>{tg.on_hand_after}</strong>
                        <span className="text-ink-4">{t('merge.cost')}</span> {money(tg.cost_before_minor_units)} <ArrowRight size={10} /> <strong>{money(tg.cost_after_minor_units)}</strong>
                      </p>
                    )
                  })}
                </div>
              )}
              {preview && !complete && <p className="text-2xs text-warning">{t('merge.matchAll')}</p>}

              <div>
                <label htmlFor="merge-confirm-name" className="block text-2xs font-medium text-ink-3 mb-1">
                  <Trans t={t} i18nKey="merge.typeToConfirm" values={{ name: targetName }} components={{ strong: <strong className="text-ink" /> }} />
                </label>
                <input id="merge-confirm-name" value={typedName} onChange={e => setTypedName(e.target.value)} className={inputClass} autoComplete="off" />
              </div>
            </>
          )}

          {loading && <p className="text-2xs text-ink-4">{t('common:status.loading')}</p>}
          {(error || mergeError) && <p role="alert" className="text-2xs text-danger">{mergeError || error}</p>}

          <div className="flex justify-end gap-2 pt-1">
            <button type="button" onClick={onClose} disabled={merging} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover disabled:opacity-50">
              {t('common:actions.cancel')}
            </button>
            <button
              type="button" onClick={handleMerge} disabled={!complete || !nameConfirmed || loading || merging}
              className="px-3 py-1.5 text-xs rounded-lg bg-red-600 text-white hover:bg-red-700 disabled:opacity-50"
            >
              {merging ? t('merge.merging') : t('merge.mergePermanently')}
            </button>
          </div>
        </div>
      </div>
      {confirmDialog}
    </Dialog>
  )
}

function skuLabel(s: MergePreviewSKU): string {
  return s.barcode ? `${s.sku_code} (${s.barcode})` : s.sku_code
}

function TargetSearch({ token, excludeId, onPick }: { token: string; excludeId: string; onPick: (id: string) => void }) {
  const { t } = useTranslation('pms')
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<Product[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (query.trim().length < 2) {
      setResults([])
      return
    }
    let cancelled = false
    const timeout = setTimeout(() => {
      api.listProducts(token, { search: query.trim(), perPage: 10 })
        .then(r => { if (!cancelled) setResults(r.products.filter(p => p.id !== excludeId)) })
        .catch((e: unknown) => { if (!cancelled) setError(e instanceof Error ? e.message : t('merge.searchFailed')) })
    }, 250)
    return () => { cancelled = true; clearTimeout(timeout) }
  }, [token, query, excludeId])

  return (
    <div className="space-y-2">
      <label htmlFor="merge-target-search" className="block text-2xs font-medium text-ink-3">{t('merge.keep')}</label>
      <input id="merge-target-search" value={query} onChange={e => setQuery(e.target.value)} placeholder={t('merge.searchPlaceholder')} className={inputClass} autoComplete="off" />
      {error && <p role="alert" className="text-2xs text-danger">{error}</p>}
      {results.length > 0 && (
        <ul className="rounded-lg border border-line divide-y divide-line/60 max-h-60 overflow-y-auto">
          {results.map(p => (
            <li key={p.id}>
              <button type="button" onClick={() => onPick(p.id)} className="w-full text-left px-2.5 py-1.5 text-xs text-ink hover:bg-hover">
                {p.name}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
